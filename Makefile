FRONTEND_DIR = webconsole/frontend
GOPATH ?= $(shell go env GOPATH)
OPI_API_URL = https://$(USER)-dev-opitransfer.sandbox.googleapis.com
OPI_ROBOT_ACCOUNT = cloud-ingest-dcp@cloud-ingest-dev.iam.gserviceaccount.com

# Add new top-level Go packages here.
GO_TARGETS = \
	./agent/... \
	./gcloud/... \
	./helpers/...

# Add individual files needing mocking here.
FILES_TO_MOCK = \
	gcloud/gcsclient.go \
	gcloud/pubsubclient.go

# NOTE: If/When we decide to move mocks to a separate directory and their own
#       packages, this will have to switch to the reflection-based mockgen.
define generate_mock
	echo "Mocking $(1)...";
	$(eval src = $(1))
	$(eval dst = $(dir $(1))mock_$(notdir $(1)))
	$(eval pkg = $(notdir $(patsubst %/,%,$(dir $(1)))))
	$(GOPATH)/bin/mockgen -source $(src) -destination $(dst) -package $(pkg);
endef

# TODO: Generate proto

.PHONY: go-mocks
go-mocks: ## Generate go mock files.
	@echo -e "\n== Generating Mocks =="
	@$(foreach file, $(FILES_TO_MOCK), $(call generate_mock,$(file)))

.PHONY: lint
lint: lint-go lint-frontend ## Run all code style validators.

.PHONY: lint-go
lint-go: ## Run Go format.
	@echo -e "\n== Formatting Go =="
	@go fmt $(GO_TARGETS)

.PHONY: lint-frontend
lint-frontend: ## Lint frontend code.
	@echo -e "\n== Running Frontend Lint =="
	@(cd $(FRONTEND_DIR) && ng lint --type-check)

.PHONY: test
test: test-go test-frontend ## Run all unit tests.

.PHONY: test-go
test-go: ## Run all go unit tests.
	@echo -e "\n== Running Go Tests =="
	@go test $(GO_TARGETS)

.PHONY: test-frontend
test-frontend: ## Run unit tests for webconsole frontend.
	@echo -e "\n== Running Frontend Tests =="
ifndef SKIP_FRONTEND_TEST
	@(cd $(FRONTEND_DIR) && OPI_API_URL=$(OPI_API_URL) OPI_ROBOT_ACCOUNT=$(OPI_ROBOT_ACCOUNT) npm test -- --watch=false)
else
	@echo -n `tput setaf 1` # Red text
	@echo "======================================"
	@echo "== WARNING: SKIPPING FRONTEND TESTS =="
	@echo "======================================"
	@echo -n `tput sgr0` # Reset
endif

.PHONY: build
build: setup build-go build-frontend ## Refresh dependencies, Build, test, and install everything.

.PHONY: build-go
build-go: go-mocks lint-go test-go ## Build, test, and install Go binaries.
	@echo -e "\n== Building/Installing Go Binaries =="
	@go install -v $(GO_TARGETS)

.PHONY: build-frontend
build-frontend: lint-frontend test-frontend ## Check and test frontend code.

# rmdir ... ;true ignores errors if dir does not exist - according to GNU make
# documentation, prefixing the line with - should accomplish this, but it doesn't work.
.PHONY: clean
clean: ## Blow away all compiled artifacts and installed dependencies.
	go clean -i $(GO_TARGETS)
	rm -rf $(FRONTEND_DIR)/node_modules release/tmp-release-ephemeral; true

.PHONY: setup
setup: setup-go setup-frontend ## Run full setup of dependencies and environment.

.PHONY: setup-go
setup-go: ## Install all needed go dependencies.
	@echo -e "\n== Installing/Updating Go Dependencies =="
	go get -u cloud.google.com/go/pubsub
	go get -u github.com/golang/glog
	go get -u github.com/golang/groupcache/lru
	go get -u github.com/golang/mock/gomock
	go get -u github.com/golang/mock/mockgen
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u golang.org/x/time/rate

.PHONY: setup-frontend
setup-frontend: ## Install all needed frontend/JS dependencies.
	@echo -e "\n== Installing/Updating Frontend Dependencies =="
	(cd $(FRONTEND_DIR) && npm install)

# Shamelessly borrowed from: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help:
	@echo $(GO_TARGETS)
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	@echo -e "\nDefault Target: $(.DEFAULT_GOAL)"
	@echo "User-supplied environment variables:"
	@echo "  SKIP_FRONTEND_TEST: If set, the frontend unit tests are skipped. Useful when"
	@echo "                      no browser is available. (default: unset)"

.DEFAULT_GOAL := build
