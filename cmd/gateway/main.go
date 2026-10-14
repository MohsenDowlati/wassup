package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/MohsenDowlati/wassup/internal/core/service"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	wsAdapter "github.com/MohsenDowlati/wassup/internal/adapter/handler/websocket"
	"github.com/MohsenDowlati/wassup/internal/adapter/secondary/mongodb"
	"github.com/MohsenDowlati/wassup/internal/adapter/secondary/redis"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
	"github.com/MohsenDowlati/wassup/internal/security"
	"github.com/MohsenDowlati/wassup/internal/telemetry"
)

func main() {
	logger := telemetry.NewProductionLogger("wassup-gateway", "Production", nil)
	metrics := telemetry.NewMetrics("wassup")

	nodeID, err := os.Hostname()
	if err != nil || nodeID == "" {
		nodeID = fmt.Sprintf("gw-node-%d", time.Now().UnixNano())
	}
	logger.Info("starting gateway node", slog.String("node_id", nodeID))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	redisClient, err := redis.NewClientFromEnv()
	if err != nil {
		logger.Error("failed to connect to redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer redisClient.Close()

	mongoClient, mongoDB, err := mongodb.NewClientFromEnv()
	if err != nil {
		logger.Error("failed to connect to mongodb", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(disconnectCtx)
	}()

	// Wires
	sessionRegistry := redis.NewSessionRegistry(redisClient)
	pubsubManager := redis.NewPubSubManager(redisClient, logger)
	defer pubsubManager.Close()

	offlineRepo := mongodb.NewOfflineRepository(mongoDB, "offline_messages")
	rateLimiter := redis.NewRedisRateLimiter(redisClient)
	hub := wsAdapter.NewHub(logger)

	dispatcher := service.NewInstrumentedDispatcher(
		sessionRegistry,
		pubsubManager,
		hub,
		offlineRepo,
		nodeID,
		7*24*time.Hour,
		metrics,
		logger,
	)

	go func() {
		err := pubsubManager.SubscribeGatewayInbox(ctx, nodeID, func(msgCtx context.Context, env *domain.Envelope) error {
			hub.RouteLocal(env.RecipientID, env)
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("pubsub subscription error", slog.String("error", err.Error()))
		}
	}()

	pasetoVerifier, err := loadPasetoVerifierFromEnv()
	if err != nil {
		logger.Error("failed to initialize paseto verifier", slog.String("error", err.Error()))
		os.Exit(1)
	}

	originValidator := security.NewOriginValidator([]string{
		"https://app.wassup.chat",
		"https://wassup.chat",
	}, true)

	upgradeHandler := wsAdapter.NewSecureUpgradeHandler(
		pasetoVerifier,
		rateLimiter,
		originValidator,
		hub,
		dispatcher,
		logger,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", upgradeHandler.ServeHTTP)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("READY"))
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		logger.Info("http server listening", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received; commencing drain")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", slog.String("error", err.Error()))
	}

	hub.BroadcastShutdown(shutdownCtx)

	logger.Info("gateway graceful shutdown completed")
}
