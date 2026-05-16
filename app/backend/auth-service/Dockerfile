# build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY backend/auth-service/go.mod    backend/auth-service/go.sum    ./backend/auth-service/
COPY backend/project-service/go.mod backend/project-service/go.sum ./backend/project-service/
COPY backend/api-gateway/go.mod     backend/api-gateway/go.sum     ./backend/api-gateway/

RUN cd backend/auth-service && go mod download

COPY backend/ ./backend/

WORKDIR /app/backend/auth-service/cmd/main
RUN go build -o /auth-service .

# runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates

COPY --from=builder /auth-service /auth-service
COPY backend/auth-service/configs/ /configs/
COPY frontend/auth/ /static/

EXPOSE 8080
ENTRYPOINT ["/auth-service", "-config", "/configs/config.docker.yaml"]
