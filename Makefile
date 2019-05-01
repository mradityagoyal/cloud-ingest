RELEASE_DIR = release
GOPATH ?= $(shell go env GOPATH)
REPO_PATH = $(GOPATH)/src/github.com/GoogleCloudPlatform/cloud-ingest
CHANGELOG_PARSER_JS = $(REPO_PATH)/node_modules/changelog-parser/bin/cli.js

# Add new top-level Go packages here.
GO_TARGETS = \
	./agent/... \
	./release/changelog/...

# Add individual files needing mocking here.
FILES_TO_MOCK = \
	agent/gcloud/gcsclient.go \
	agent/pubsub/pubsubclient.go \
	$(GOPATH)/src/github.com/googleapis/google-cloud-go-testing/storage/stiface/interfaces.go

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
lint: lint-agent lint-changelog ## Run all code style validators.

.PHONY: lint-agent
lint-agent: ## Run Go format.
	@echo -e "\n== Formatting Go =="
	@go fmt $(GO_TARGETS)

.PHONY: lint-changelog
lint-changelog: ## Validate changelog format.
	@echo -e "\n== Validating Changelog Format =="
	@go run "$(RELEASE_DIR)/validatechangelog.go" -buildType dev

.PHONY: test
test: test-agent ## Run all unit tests.

.PHONY: test-agent
test-agent: go-mocks ## Run all go unit tests.
	@echo -e "\n== Running Go Tests =="
	@go test $(GO_TARGETS)

.PHONY: build
build: setup build-agent ## Refresh dependencies, Build, test, and install everything.

.PHONY: build-agent
build-agent: install-changelog-parser go-mocks lint-agent lint-changelog test-agent ## Build, test, and install Go binaries.
	@echo -e "\n== Building/Installing Go Binaries =="
	@go install -v $(GO_TARGETS)

.PHONY: validate-release-changelog
validate-release-changelog: ## Validate changelog format and new release version.
	@echo -e "\n== Validating Changelog Format And Release Version =="
	@go run "$(RELEASE_DIR)/validatechangelog.go" -buildType prod

.PHONY: build-release-agent
build-release-agent: go-mocks lint-agent validate-release-changelog test-agent ## Build, test, and install Go binaries.
	@echo -e "\n== Building/Installing Go Binaries =="
	@go install -v $(GO_TARGETS)

# rmdir ... ;true ignores errors if dir does not exist - according to GNU make
# documentation, prefixing the line with - should accomplish this, but it doesn't work.
.PHONY: clean
clean: ## Blow away all compiled artifacts and installed dependencies.
	go clean -i $(GO_TARGETS)
	rm -rf node_modules $(RELEASE_DIR)/tmp-release-ephemeral; true

.PHONY: setup
setup: setup-agent ## Run full setup of dependencies and environment.

.PHONY: setup-agent
setup-agent: pull-agent-go-dependencies install-changelog-parser ## Install all needed agent dependencies.

.PHONY: pull-agent-go-dependencies
pull-agent-go-dependencies: ## Pull all go library dependencies needed for building the agent.
	@echo -e "\n== Installing/Updating Go Dependencies =="
	go get -u cloud.google.com/go/pubsub
	go get -u github.com/blang/semver
	go get -u github.com/golang/glog
	go get -u github.com/golang/groupcache/lru
	go get -u github.com/golang/mock/gomock
	go get -u github.com/golang/mock/mockgen
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u github.com/google/go-cmp/cmp
	go get -u github.com/googleapis/google-cloud-go-testing
	go get -u golang.org/x/time/rate

.PHONY: install-changelog-parser
install-changelog-parser: ## Install the changelog parser.
	@echo -e "\n== Installing Changelog Parser =="
	@(test -f $(CHANGELOG_PARSER_JS) && echo "Already installed...") || npm install changelog-parser --loglevel error

# Shamelessly borrowed from: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help:
	@echo $(GO_TARGETS)
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	@echo -e "\nDefault Target: $(.DEFAULT_GOAL)"

.DEFAULT_GOAL := build
