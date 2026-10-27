# Lesson 13 (gRPC microservices). The targets grow with the parts of the homework; `make` lists them.
.DEFAULT_GOAL := help

help:          ## list the targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

tools:         ## install the protoc plugins at the versions pinned by the course demo
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

proto:         ## regenerate gen/albums from proto/albums.proto (needs protoc and the plugins in PATH)
	protoc -I proto \
	  --go_out=. --go_opt=module=homework \
	  --go-grpc_out=. --go-grpc_opt=module=homework \
	  proto/albums.proto

build:         ## build the server and the client into bin/
	go build -o bin/ ./cmd/albums-server ./cmd/albums-client

server:        ## run the gRPC server on :50051 (plaintext, local development only)
	go run ./cmd/albums-server

client:        ## run the client, for example: make client ARGS="get 1"
	go run ./cmd/albums-client $(ARGS)

test:          ## race tests of the own module with a coverage profile (cover.out)
	go test -race -count=1 -coverprofile=cover.out ./...

smoke:         ## end-to-end smoke check of the own service (builds bin/, server on 127.0.0.1:50061)
	bash scripts/smoke.sh

demo-test:     ## race tests of the course demo
	cd libratrack-grpc-demo && go test ./... -race -count=1

demo-smoke:    ## end-to-end smoke check of the course demo
	cd libratrack-grpc-demo && bash scripts/smoke.sh

.PHONY: help tools proto build server client test smoke demo-test demo-smoke
