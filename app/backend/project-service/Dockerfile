# build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY backend/auth-service/go.mod backend/auth-service/go.sum ./backend/auth-service/
COPY backend/project-service/go.mod backend/project-service/go.sum ./backend/project-service/

RUN cd backend/auth-service && go mod download && cd /app/backend/project-service && go mod download

COPY backend/ ./backend/

WORKDIR /app/backend/project-service/cmd/main
RUN go build -o /project-service .

# runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates

COPY --from=builder /project-service /project-service
COPY backend/project-service/configs/ /configs/
COPY frontend/project/ /static/

EXPOSE 8081
ENTRYPOINT ["/project-service", "-config", "/configs/config.docker.yaml"]
