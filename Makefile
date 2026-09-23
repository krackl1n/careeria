SHELL := /bin/sh

.DEFAULT_GOAL := help

GO ?= go

COMPOSE_LOCAL ?= docker compose -f docker-compose.local.yaml
COMPOSE_SERVER ?= docker compose -f docker-compose.server.yaml
COMPOSE ?= $(COMPOSE_LOCAL)

GO_TOOL_BIN := $(shell $(GO) env GOBIN)
ifeq ($(strip $(GO_TOOL_BIN)),)
GO_TOOL_BIN := $(shell $(GO) env GOPATH)/bin
endif

BUF_VERSION ?= v1.69.0
GOOSE_VERSION ?= v3.27.2
PROTOC_GEN_GO_VERSION ?= v1.36.11
PROTOC_GEN_GO_GRPC_VERSION ?= v1.5.1
GRPC_GATEWAY_VERSION ?= v2.29.0
PROTOC_GEN_VALIDATE_VERSION ?= v1.3.3

OPENFGA_CLI_VERSION ?= v0.7.20
OPENFGA_CLI_IMAGE ?= openfga/cli:$(OPENFGA_CLI_VERSION)
OPENFGA_CLI_API_URL ?= http://host.docker.internal:$(OPENFGA_HTTP_PORT)
OPENFGA_HTTP_PORT ?= 18081
OPENFGA_STORE_NAME ?= careeria

STRUCTURIZR_VERSION ?= 2026.09.19
STRUCTURIZR_IMAGE ?= structurizr/structurizr:$(STRUCTURIZR_VERSION)
STRUCTURIZR_PORT ?= 18882
PLANTUML_VERSION ?= 1.2026.8
PLANTUML_IMAGE ?= plantuml/plantuml:$(PLANTUML_VERSION)

BUF ?= $(GO_TOOL_BIN)/buf
GOOSE ?= $(GO_TOOL_BIN)/goose

CONTRACTS_DIR := contracts
PROTO_DIR := $(CONTRACTS_DIR)/proto
PROTO_VENDOR_DIR := $(CONTRACTS_DIR)/vendor

OPENFGA_MODEL_DIR := $(CURDIR)/authz/openfga
OPENFGA_MODEL_FILE := /model/model.fga
OPENFGA_TEST_FILE := /model/model.fga.yaml

STRUCTURIZR_DIR := $(CURDIR)/docs/architecture/c4
STRUCTURIZR_WORKSPACE := /usr/local/structurizr/workspace.dsl
PLANTUML_ARCHITECTURE_DIR := docs/architecture/c4-plant-uml

MODULE_DIRS := $(shell find . -name go.mod -not -path '*/vendor/*' -exec dirname {} \; | sort)
MIGRATION_SERVICES := identity

.PHONY: \
	help \
	bootstrap deps tools proto-tools install-goose \
	proto-deps proto-deps-check proto-vendor proto-vendor-check \
	up up-d up-server up-server-d \
	server-openfga-bootstrap local-bootstrap local-bootstrap-roles \
	down down-server clean restart logs ps \
	docker-build compose-config compose-config-server \
	build test vet check fmt \
	authz-model-test authz-store-create authz-model-write \
	architecture architecture-validate plantuml-architecture plantuml-architecture-validate \
	run-gateway run-identity \
	migrate-new migrate-up migrate-status migrate-version \
	proto-generate proto-lint proto-format

help: ## Print available commands.
	@awk 'BEGIN {FS = ":.*##"; print "Usage: make <target>\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-24s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

bootstrap: deps tools ## Prepare a development machine without changing vendored proto dependencies.

deps: ## Download every Go module dependency.
	@set -e; \
	for module in $(MODULE_DIRS); do \
		(cd "$$module" && GOWORK=off $(GO) mod download); \
	done

tools: proto-tools install-goose ## Install all development tools.

proto-tools: ## Install tools required for protobuf generation.
	$(GO) install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)
	$(GO) install github.com/envoyproxy/protoc-gen-validate@$(PROTOC_GEN_VALIDATE_VERSION)
	$(GO) install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@$(GRPC_GATEWAY_VERSION)
	$(GO) install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@$(GRPC_GATEWAY_VERSION)

install-goose: ## Install Goose using go install.
	$(GO) install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)

proto-deps: proto-vendor-check ## Compatibility alias: verify locally vendored protobuf dependencies.

proto-deps-check: proto-vendor-check ## Compatibility alias for offline protobuf dependency verification.

proto-vendor: proto-vendor-check ## Verify protobuf dependencies vendored in the repository.

proto-vendor-check: ## Verify files required for offline protobuf generation.
	@test -f "$(PROTO_VENDOR_DIR)/manifest.yaml"
	@test -f "$(PROTO_VENDOR_DIR)/envoyproxy/protoc-gen-validate/validate/validate.proto"
	@test -f "$(PROTO_VENDOR_DIR)/googleapis/googleapis/google/api/annotations.proto"
	@test -f "$(PROTO_VENDOR_DIR)/grpc-ecosystem/grpc-gateway/protoc-gen-openapiv2/options/openapiv2.proto"

up: ## Start the local stack in the foreground.
	$(COMPOSE) up --build

up-d: ## Start the local stack in the background.
	$(COMPOSE) up --build --detach

up-server: ## Start the server stack in the foreground (only Web apps are public).
	$(COMPOSE_SERVER) up --build

up-server-d: ## Start the server stack in the background (only Web apps are public).
	$(COMPOSE_SERVER) up --build --detach

server-openfga-bootstrap: ## Create or reuse the server OpenFGA store, publish the current model, and update .env IDs.
	./scripts/openfga/bootstrap-server.sh

local-bootstrap: ## Start and initialize the complete local platform interactively.
	./scripts/bootstrap-local.sh

local-bootstrap-roles: ## Create missing baseline roles; ORGANIZATION_ID and OWNER_SUB are required.
	@test -n "$(ORGANIZATION_ID)" || { \
		printf '%s\n' 'ORGANIZATION_ID is required' >&2; \
		exit 2; \
	}
	@test -n "$(OWNER_SUB)" || { \
		printf '%s\n' 'OWNER_SUB is required' >&2; \
		exit 2; \
	}
	./scripts/organization/bootstrap-roles.sh \
		--organization-id "$(ORGANIZATION_ID)" \
		--owner-sub "$(OWNER_SUB)"

down: ## Stop and remove local containers and the network.
	$(COMPOSE) down

down-server: ## Stop and remove the server Compose stack.
	$(COMPOSE_SERVER) down

clean: ## Stop the stack and remove local PostgreSQL data.
	$(COMPOSE) down --volumes --remove-orphans

restart: ## Restart running local services.
	$(COMPOSE) restart

logs: ## Follow local service logs.
	$(COMPOSE) logs --follow

ps: ## Show local service status.
	$(COMPOSE) ps

docker-build: ## Build Docker images without starting services.
	$(COMPOSE) build

compose-config: ## Validate and render the local Compose configuration.
	$(COMPOSE) config

compose-config-server: ## Validate and render the server Compose configuration.
	$(COMPOSE_SERVER) config

build: ## Compile every Go module.
	@set -e; \
	for module in $(MODULE_DIRS); do \
		(cd "$$module" && $(GO) build ./...); \
	done

test: ## Run tests in every Go module.
	@set -e; \
	for module in $(MODULE_DIRS); do \
		(cd "$$module" && $(GO) test ./...); \
	done

vet: ## Run go vet in every Go module.
	@set -e; \
	for module in $(MODULE_DIRS); do \
		(cd "$$module" && $(GO) vet ./...); \
	done

check: test vet proto-lint authz-model-test architecture-validate ## Run the local verification suite.

fmt: ## Format all Go source files.
	@gofmt -w $$(find . -name '*.go' -not -path '*/vendor/*')

authz-model-test: ## Validate the OpenFGA model and run its authorization tests.
	docker run --rm \
		-v "$(OPENFGA_MODEL_DIR):/model:ro" \
		$(OPENFGA_CLI_IMAGE) \
		model test --tests $(OPENFGA_TEST_FILE)

authz-store-create: ## Create an OpenFGA store with the current model and print both generated IDs.
	docker run --rm \
		--add-host host.docker.internal:host-gateway \
		-e FGA_API_URL="$(OPENFGA_CLI_API_URL)" \
		-v "$(OPENFGA_MODEL_DIR):/model:ro" \
		$(OPENFGA_CLI_IMAGE) \
		store create \
			--name "$(OPENFGA_STORE_NAME)" \
			--model $(OPENFGA_MODEL_FILE)

authz-model-write: ## Write a new immutable model version; OPENFGA_STORE_ID is required.
	@test -n "$(OPENFGA_STORE_ID)" || { \
		printf '%s\n' 'OPENFGA_STORE_ID is required' >&2; \
		exit 2; \
	}
	docker run --rm \
		--add-host host.docker.internal:host-gateway \
		-e FGA_API_URL="$(OPENFGA_CLI_API_URL)" \
		-v "$(OPENFGA_MODEL_DIR):/model:ro" \
		$(OPENFGA_CLI_IMAGE) \
		model write \
			--store-id "$(OPENFGA_STORE_ID)" \
			--file $(OPENFGA_MODEL_FILE)

architecture: ## Start the local Structurizr architecture viewer.
	@test -f "$(STRUCTURIZR_DIR)/workspace.dsl" || { \
		printf '%s\n' 'Structurizr workspace not found: $(STRUCTURIZR_DIR)/workspace.dsl' >&2; \
		exit 2; \
	}
	docker run --rm -it \
		-p "$(STRUCTURIZR_PORT):8080" \
		-v "$(STRUCTURIZR_DIR):/usr/local/structurizr" \
		$(STRUCTURIZR_IMAGE) \
		local

architecture-validate: ## Validate the Structurizr architecture workspace.
	@test -f "$(STRUCTURIZR_DIR)/workspace.dsl" || { \
		printf '%s\n' 'Structurizr workspace not found: $(STRUCTURIZR_DIR)/workspace.dsl' >&2; \
		exit 2; \
	}
	docker run --rm \
		-v "$(STRUCTURIZR_DIR):/usr/local/structurizr:ro" \
		$(STRUCTURIZR_IMAGE) \
		validate -workspace $(STRUCTURIZR_WORKSPACE)

plantuml-architecture: ## Render C4-PlantUML architecture diagrams to SVG.
	docker run --rm \
		-v "$(CURDIR):/workspace" \
		-w /workspace \
		$(PLANTUML_IMAGE) \
		-tsvg -o ../out $(PLANTUML_ARCHITECTURE_DIR)/views/*.puml

plantuml-architecture-validate: ## Validate all C4-PlantUML architecture diagrams.
	docker run --rm \
		-v "$(CURDIR):/workspace:ro" \
		-w /workspace \
		$(PLANTUML_IMAGE) \
		-checkonly $(PLANTUML_ARCHITECTURE_DIR)/views/*.puml

run-gateway: ## Run API Gateway from source.
	$(GO) -C services/api-gateway run ./cmd/gateway

run-identity: ## Run Identity from source.
	$(GO) -C services/identity run ./cmd/service

migrate-new: install-goose ## Create a Goose SQL migration; SERVICE and NAME are required.
	@case "$(SERVICE)" in \
		identity) ;; \
		*) \
			printf '%s\n' 'SERVICE must be one of: $(MIGRATION_SERVICES)' >&2; \
			exit 2; \
			;; \
	esac
	@test -n "$(NAME)" || { \
		printf '%s\n' 'NAME is required, for example NAME=create_meetings' >&2; \
		exit 2; \
	}
	@mkdir -p services/$(SERVICE)/db/migrations
	$(GOOSE) \
		-dir services/$(SERVICE)/db/migrations \
		create $(NAME) sql

migrate-up: ## Apply migrations through the service Goose runner; SERVICE is required.
	$(call require_migration_service)
	$(GO) -C services/$(SERVICE) run ./cmd/migrate up

migrate-status: ## Show migration status through the service Goose runner; SERVICE is required.
	$(call require_migration_service)
	$(GO) -C services/$(SERVICE) run ./cmd/migrate status

migrate-version: ## Show the current migration version; SERVICE is required.
	$(call require_migration_service)
	$(GO) -C services/$(SERVICE) run ./cmd/migrate version

define require_migration_service
	@case "$(SERVICE)" in \
		identity|meeting|document|organization) ;; \
		*) \
			printf '%s\n' 'SERVICE must be one of: $(MIGRATION_SERVICES)' >&2; \
			exit 2; \
			;; \
	esac
endef

proto-generate: proto-tools proto-vendor-check ## Generate Go protobuf and gRPC code using vendored dependencies.
	@if test -z "$$(find "$(PROTO_DIR)" -name '*.proto' -print -quit)"; then \
		printf '%s\n' 'No .proto files found under $(PROTO_DIR); generation skipped.'; \
	else \
		(cd "$(CONTRACTS_DIR)" && PATH="$(GO_TOOL_BIN):$$PATH" $(BUF) generate); \
		GOWORK=off $(GO) -C packages/go/contracts run ./cmd/openapi-normalize \
			-file openapi/careeria.swagger.json; \
		GOWORK=off $(GO) -C packages/go/contracts mod tidy; \
	fi

proto-lint: proto-tools proto-vendor-check ## Lint protobuf files using vendored dependencies.
	@if test -z "$$(find "$(PROTO_DIR)" -name '*.proto' -print -quit)"; then \
		printf '%s\n' 'No .proto files found under $(PROTO_DIR); lint skipped.'; \
	else \
		(cd "$(CONTRACTS_DIR)" && $(BUF) lint); \
	fi

proto-format: proto-tools ## Format protobuf files with Buf.
	@if test -z "$$(find "$(PROTO_DIR)" -name '*.proto' -print -quit)"; then \
		printf '%s\n' 'No .proto files found under $(PROTO_DIR); formatting skipped.'; \
	else \
		(cd "$(CONTRACTS_DIR)" && $(BUF) format --write); \
	fi
