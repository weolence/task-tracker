FROM golang:1.26-alpine

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o ./project-service ./cmd/main

EXPOSE 8081
CMD ["./project-service"]
