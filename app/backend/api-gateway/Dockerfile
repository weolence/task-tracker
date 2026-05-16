# build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY backend/auth-service/go.mod    backend/auth-service/go.sum    ./backend/auth-service/
COPY backend/project-service/go.mod backend/project-service/go.sum ./backend/project-service/
COPY backend/api-gateway/go.mod     backend/api-gateway/go.sum     ./backend/api-gateway/

RUN cd backend/api-gateway && go mod download

COPY backend/ ./backend/

WORKDIR /app/backend/api-gateway/cmd/main
RUN go build -o /api-gateway .

# runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates

COPY --from=builder /api-gateway /api-gateway
COPY backend/api-gateway/configs/ /configs/

EXPOSE 8000
ENTRYPOINT ["/api-gateway", "-config", "/configs/config.docker.yaml"]
