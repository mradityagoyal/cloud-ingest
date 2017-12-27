FRONTEND_DIR = webconsole/frontend
BACKEND_DIR = webconsole/backend
OPI_BACKEND_VIRTUALENV_PATH ?= ~/cloud-ingest-backend-env
FULL_OPI_BACKEND_VIRTUALENV_PATH = $(OPI_BACKEND_VIRTUALENV_PATH)/opi-virtualenv

# Add new top-level Go packages here.
GO_TARGETS = \
	./agent/... \
	./dcp/... \
	./gcloud/... \
	./helpers/... \
	./tests/...

# Add individual files needing mocking here.
FILES_TO_MOCK = \
	dcp/listresultreader.go \
	dcp/objectmetadatareader.go \
	gcloud/gcsclient.go \
	gcloud/pubsubclient.go \
	gcloud/spannerclient.go \
	helpers/clock.go \
	helpers/random.go \
	tests/perf/jobservice.go

# NOTE: If/When we decide to move mocks to a separate directory and their own
#       packages, this will have to switch to the reflection-based mockgen.
define generate_mock
	echo "Mocking $(1)...";
	$(eval src = $(1))
	$(eval dst = $(dir $(1))mock_$(notdir $(1)))
	$(eval pkg = $(notdir $(patsubst %/,%,$(dir $(1)))))
	mockgen -source $(src) -destination $(dst) -package $(pkg);
endef

# TODO: Generate proto

.PHONY: go-mocks
go-mocks: ## Generate go mock files.
	@echo -e "\n== Generating Mocks =="
	@$(foreach file, $(FILES_TO_MOCK), $(call generate_mock,$(file)))

.PHONY: lint
lint: lint-go lint-backend lint-frontend ## Run all code style validators.

.PHONY: lint-go
lint-go: ## Run Go format.
	@echo -e "\n== Formatting Go =="
	@go fmt $(GO_TARGETS)

.PHONY: lint-backend
lint-backend: ## Lint backend code.
	@echo -e "\n== Running Backend Lint =="
	@find $(BACKEND_DIR) -type f -name "*.py" | egrep -ve node_modules | \
		xargs pylint --rcfile=.pylintrc

.PHONY: lint-frontend
lint-frontend: ## Lint frontend code.
	@echo -e "\n== Running Frontend Lint =="
	@(cd $(FRONTEND_DIR) && ng lint --type-check)

.PHONY: test
test: test-go test-backend test-frontend ## Run all unit tests.

.PHONY: test-go
test-go: ## Run all go unit tests.
	@echo -e "\n== Running Go Tests =="
	@go test $(GO_TARGETS)

.PHONY: test-backend
test-backend: ## Backend unit tests.
	@echo -e "\n== Running Backend Tests =="
	@( \
	    source $(FULL_OPI_BACKEND_VIRTUALENV_PATH)/bin/activate; \
	    test $$VIRTUAL_ENV && INGEST_CONFIG_PATH=ingestwebconsole.test_settings python -m unittest discover $(BACKEND_DIR) && deactivate; \
	)

.PHONY: test-frontend
test-frontend: ## Run unit tests for webconsole frontend.
	@echo -e "\n== Running Frontend Tests =="
ifndef SKIP_FRONTEND_TEST
	@(cd $(FRONTEND_DIR) && ng test --watch=false)
else
	@echo -n `tput setaf 1` # Red text
	@echo "======================================"
	@echo "== WARNING: SKIPPING FRONTEND TESTS =="
	@echo "======================================"
	@echo -n `tput sgr0` # Reset
endif

.PHONY: build
build: setup build-go build-backend build-frontend ## Refresh dependencies, Build, test, and install everything.

.PHONY: build-go
build-go: lint-go go-mocks test-go ## Build, test, and install Go binaries.
	@echo -e "\n== Building/Installing Go Binaries =="
	@go install -v $(GO_TARGETS)

.PHONY: build-backend
build-backend: lint-backend test-backend ## Check and test backend code.

.PHONY: build-frontend
build-frontend: lint-frontend test-frontend ## Check and test frontend code.

# rmdir ... ;true ignores errors if dir does not exist - according to GNU make
# documentation, prefixing the line with - should accomplish this, but it doesn't work.
.PHONY: clean
clean: ## Blow away all compiled artifacts and installed dependencies.
	go clean -i $(GO_TARGETS)
	rm -rf $(FRONTEND_DIR)/node_modules
	rm -rf $(FULL_OPI_BACKEND_VIRTUALENV_PATH)
	rmdir --ignore-fail-on-non-empty $(OPI_BACKEND_VIRTUALENV_PATH); true

.PHONY: setup
setup: setup-go setup-backend setup-frontend ## Run full setup of dependencies and environment.

.PHONY: setup-go
setup-go: ## Install all needed go dependencies.
	@echo -e "\n== Installing/Updating Go Dependencies =="
	go get -u cloud.google.com/go/pubsub
	go get -u cloud.google.com/go/spanner
	go get -u github.com/golang/glog
	go get -u github.com/golang/groupcache/lru
	go get -u github.com/golang/mock/gomock
	go get -u github.com/golang/mock/mockgen
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u golang.org/x/time/rate

.PHONY: setup-backend
setup-backend: ## Install all needed backend dependencies.
	@echo -e "\n== Installing/Updating Backend Dependencies =="
	@mkdir -p $(FULL_OPI_BACKEND_VIRTUALENV_PATH)
	@virtualenv $(FULL_OPI_BACKEND_VIRTUALENV_PATH)
	@( \
	    source $(FULL_OPI_BACKEND_VIRTUALENV_PATH)/bin/activate; \
	    test $$VIRTUAL_ENV && pip install -r requirements.txt && deactivate; \
	)

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
	@echo "  OPI_BACKEND_VIRTUALENV_PATH: Location where backend python virtualenv should"
	@echo "                               live (default: $(OPI_BACKEND_VIRTUALENV_PATH))"
	@echo "  SKIP_FRONTEND_TEST: If set, the frontend unit tests are skipped. Useful when"
	@echo "                      no browser is available. (default: unset)"

.DEFAULT_GOAL := build
