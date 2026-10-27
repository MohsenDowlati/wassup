package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
	ports "github.com/MohsenDowlati/wassup/internal/core/ports"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	maxMessageSize = 64 * 1024

	writeTimeout = 5 * time.Second

	pingPeriod = 25 * time.Second

	outboxBufferSize = 256
)

type ClientSession struct {
	userID    string
	sessionID string
	conn      *websocket.Conn
	hub       ports.LocalConnectionHub
	router    ports.InboundMessageRouter
	logger    *slog.Logger

	outbox chan *domain.Envelope

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	closeOnce sync.Once
}

func NewClientSession(
	parentCtx context.Context,
	userID string,
	sessionID string,
	conn *websocket.Conn,
	hub ports.LocalConnectionHub,
	router ports.InboundMessageRouter,
	logger *slog.Logger,
) *ClientSession {
	ctx, cancel := context.WithCancel(parentCtx)

	conn.SetReadLimit(maxMessageSize)

	return &ClientSession{
		userID:    userID,
		sessionID: sessionID,
		conn:      conn,
		hub:       hub,
		router:    router,
		logger:    logger.With("user_id", userID, "session_id", sessionID),
		outbox:    make(chan *domain.Envelope, outboxBufferSize),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (s *ClientSession) UserID() string    { return s.userID }
func (s *ClientSession) SessionID() string { return s.sessionID }

func (s *ClientSession) Start() {
	s.wg.Add(2)
	go s.writePump()
	go s.readPump()
}

func (s *ClientSession) Enqueue(env *domain.Envelope) bool {
	select {
	case <-s.ctx.Done():
		return false
	case s.outbox <- env:
		return true
	default:
		s.logger.Warn("slow client detected: outbox buffer saturated, terminating connection")
		s.Close(int(websocket.StatusPolicyViolation), "client outbox buffer saturated")
		return false
	}
}

func (s *ClientSession) readPump() {
	defer func() {
		s.Close(int(websocket.StatusNormalClosure), "read pump exited")
		s.wg.Done()
	}()

	for {
		typ, data, err := s.conn.Read(s.ctx)
		if err != nil {
			var closeErr websocket.CloseError
			if errors.As(err, &closeErr) {
				s.logger.Debug("clean client disconnect", "code", closeErr.Code, "reason", closeErr.Reason)
			} else if !errors.Is(err, context.Canceled) {
				s.logger.Debug("socket read terminated", "err", err)
			}
			return
		}

		if typ != websocket.MessageText && typ != websocket.MessageBinary {
			s.logger.Warn("unsupported frame type received, skipping", "type", typ)
			continue
		}

		var env domain.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			s.logger.Warn("malformed payload received", "err", err)
			continue
		}

		env.SenderID = s.userID

		if err := env.Validate(); err != nil {
			s.logger.Warn("invalid envelope rejected", "err", err)
			continue
		}

		if err := s.router.RouteInbound(s.ctx, &env); err != nil {
			s.logger.Error("failed to route inbound envelope", "err", err, "idempotency_id", env.IdempotencyID)
		}
	}
}

func (s *ClientSession) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		s.Close(int(websocket.StatusNormalClosure), "write pump exited")
		s.wg.Done()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return

		case <-ticker.C:
			pingCtx, pingCancel := context.WithTimeout(s.ctx, writeTimeout)
			err := s.conn.Ping(pingCtx)
			pingCancel()
			if err != nil {
				s.logger.Debug("heartbeat ping failed", "err", err)
				return
			}

		case env, ok := <-s.outbox:
			if !ok {
				return
			}

			writeCtx, writeCancel := context.WithTimeout(s.ctx, writeTimeout)
			payload, err := json.Marshal(env)
			if err != nil {
				writeCancel()
				s.logger.Error("failed to marshal envelope", "err", err)
				continue
			}

			err = s.conn.Write(writeCtx, websocket.MessageText, payload)
			writeCancel()
			if err != nil {
				s.logger.Debug("socket write failed", "err", err)
				return
			}
		}
	}
}

func (s *ClientSession) Close(code int, reason string) {
	s.closeOnce.Do(func() {
		s.cancel()

		_ = s.conn.Close(websocket.StatusCode(code), reason)

		s.hub.Unregister(s.userID, s.sessionID)
	})
}

func (s *ClientSession) Wait() {
	s.wg.Wait()
}
