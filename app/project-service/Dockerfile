# build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY auth-service/go.mod auth-service/go.sum ./auth-service/
COPY project-service/go.mod project-service/go.sum ./project-service/

RUN cd auth-service && go mod download && cd ../project-service && go mod download

COPY auth-service/ ./auth-service/
COPY project-service/ ./project-service/

WORKDIR /app/project-service/cmd/main
RUN go build -o /project-service .

# runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates

COPY --from=builder /project-service /project-service
COPY project-service/configs/ /configs/

EXPOSE 8081
ENTRYPOINT ["/project-service", "-config", "/configs/config.docker.yaml"]
