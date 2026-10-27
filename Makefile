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

demo-test:     ## race tests of the course demo
	cd libratrack-grpc-demo && go test ./... -race -count=1

demo-smoke:    ## end-to-end smoke check of the course demo
	cd libratrack-grpc-demo && bash scripts/smoke.sh

.PHONY: help tools proto demo-test demo-smoke
