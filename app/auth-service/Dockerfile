# build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
WORKDIR /app/cmd/server
RUN go build -o /auth-service .

# runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates

COPY --from=builder /auth-service /auth-service
EXPOSE 8080
ENTRYPOINT ["/auth-service"]
