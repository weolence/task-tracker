PROTO_AUTH    := backend/auth-service/api/proto
PROTO_PROJECT := backend/project-service/api/proto

.PHONY: proto build tidy

## proto: regenerate gRPC Go code from .proto files
proto:
	@echo "→ generating auth-service gRPC stubs"
	protoc \
		-I $(PROTO_AUTH) \
		--go_out=$(PROTO_AUTH)/authsvcv1     --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_AUTH)/authsvcv1 --go-grpc_opt=paths=source_relative \
		$(PROTO_AUTH)/auth_service.proto

	@echo "→ generating project-service gRPC stubs"
	protoc \
		-I $(PROTO_AUTH) \
		-I $(PROTO_PROJECT) \
		--go_out=$(PROTO_PROJECT)/projectsvcv1     --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_PROJECT)/projectsvcv1 --go-grpc_opt=paths=source_relative \
		$(PROTO_PROJECT)/project_service.proto

## build: build all modules in the workspace
build:
	go build ./backend/auth-service/...
	go build ./backend/project-service/...
	go build ./backend/api-gateway/...

## tidy: tidy all modules
tidy:
	cd backend/auth-service    && go mod tidy
	cd backend/project-service && go mod tidy
	cd backend/api-gateway     && go mod tidy
