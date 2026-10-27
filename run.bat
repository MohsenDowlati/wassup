@echo off
setlocal

echo [1/3] Starting Infrastructure via Docker Compose...
docker compose up -d

echo [2/3] Waiting for Redis and Mongo to be ready...
timeout /t 3 /nobreak >nul

echo [3/3] Launching Gateway Node 01 on port 8081...
start "Wassup Gateway 01 (Port 8081)" cmd /k "set APP_ENV=development& set SERVER_PORT=8081& set NODE_ID=node-gw-01& set REDIS_ADDR=localhost:6379& set MONGO_URI=mongodb://localhost:27017/wassup_chat& set MONGO_DATABASE=wassup_chat& set ALLOWED_ORIGINS=http://localhost:8080,http://127.0.0.1:8080& set PASETO_PUBLIC_KEY_HEX=062b6295fa290f425c1f93f349f75e978d8f4db065c3930db6f9d319b9da1360& set PASETO_ISSUER=wassup-auth-service& set PASETO_AUDIENCE=wassup-gateway& go run ./cmd/gateway/main.go"

echo Launching Gateway Node 02 on port 8082...
start "Wassup Gateway 02 (Port 8082)" cmd /k "set APP_ENV=development& set SERVER_PORT=8082& set NODE_ID=node-gw-02& set REDIS_ADDR=localhost:6379& set MONGO_URI=mongodb://localhost:27017/wassup_chat& set MONGO_DATABASE=wassup_chat& set ALLOWED_ORIGINS=http://localhost:8080,http://127.0.0.1:8080& set PASETO_PUBLIC_KEY_HEX=062b6295fa290f425c1f93f349f75e978d8f4db065c3930db6f9d319b9da1360& set PASETO_ISSUER=wassup-auth-service& set PASETO_AUDIENCE=wassup-gateway& go run ./cmd/gateway/main.go"

echo ========================================================
echo All services started successfully!
echo Open http://localhost:8080 in your browser.
echo ========================================================
endlocal
