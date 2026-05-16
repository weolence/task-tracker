# build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
WORKDIR /app/cmd/main
RUN go build -o /project-service .

# runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates

COPY --from=builder /project-service /project-service
COPY configs/ /configs/

EXPOSE 8081
ENTRYPOINT ["/project-service", "-config", "/configs/config.docker.yaml"]
