# -------- Settings --------
PROTOC ?= protoc

# Paths
GO_GEN_DIR   = ./gen/go
GO_OUT_DIR   = ../backend/pkg/proto
TS_GEN_DIR   = ./gen/ts
TS_OUT_DIR   = ../frontend/src/app/global/model

# Protos
PROTO_FILES := $(wildcard *.proto)
PROTO_PATH  := $(shell pwd)
PROTOC_GEN_TS_PROTO ?= /usr/lib/node_modules/ts-proto/protoc-gen-ts_proto

# -------- Targets --------
.PHONY: generate proto proto-go proto-ts clean

proto: pre-gen proto-go proto-ts post-gen clean

# Run the generation in a Docker container to ensure consistent tooling across environments.
# Only for direct evaluation during development; CI pipelines should run the generation on commit for better performance and documentation.
generate:
	docker run --rm \
	  -v $(shell git rev-parse --show-toplevel):/app/ \
	  -w /app/proto \
	  ghcr.io/valentin-kaiser/protocdock:latest \
	  make proto

proto-go:
	@mkdir -p $(GO_GEN_DIR)
	$(PROTOC) -I $(PROTO_PATH) \
	  --go_out=$(GO_GEN_DIR) \
	  --go_opt=module=github.com/valentin-kaiser/hdns/backend/pkg/proto \
	  --go-jrpc_out=$(GO_GEN_DIR) \
	  --go-jrpc_opt=module=github.com/valentin-kaiser/hdns/backend/pkg/proto \
	  $(PROTO_FILES)

proto-ts:
	@mkdir -p $(TS_GEN_DIR)
	$(PROTOC) -I $(PROTO_PATH) \
	  --plugin=$(PROTOC_GEN_TS_PROTO) \
	  --ts_proto_out=$(TS_GEN_DIR) \
	  --ts_proto_opt=esModuleInterop=true,useJsonName=true,outputServices=generic-definitions,exportCommonSymbols=false \
	  $(PROTO_FILES)

pre-gen:
	@rm -rf $(GO_GEN_DIR)
	@rm -rf $(TS_GEN_DIR)
	@rm -rf $(GO_OUT_DIR)
	@rm -rf $(TS_OUT_DIR)

post-gen:
	@mkdir -p $(GO_OUT_DIR)
	@mkdir -p $(TS_OUT_DIR)
	cp -r $(GO_GEN_DIR)/* $(GO_OUT_DIR)/
	cp -r $(TS_GEN_DIR)/* $(TS_OUT_DIR)/


clean:
	rm -rf $(GO_GEN_DIR)
	rm -rf $(TS_GEN_DIR)