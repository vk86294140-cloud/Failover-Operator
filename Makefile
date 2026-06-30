IMG ?= failover-operator:latest

.PHONY: tidy
tidy: ## Resolve and pin module dependencies
	go mod tidy

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: build
build: ## Build all packages
	go build ./...

.PHONY: test
test: ## Run unit tests (no cluster needed)
	go test ./... -count=1

.PHONY: test-integration
test-integration: ## Run envtest integration tests (needs KUBEBUILDER_ASSETS via setup-envtest)
	go test -tags=integration ./internal/controller/... -count=1

.PHONY: charm-pack
charm-pack: ## Pack the Juju charm (requires charmcraft)
	cd charm && charmcraft pack

.PHONY: snap-build
snap-build: ## Build the snap (requires snapcraft)
	snapcraft

.PHONY: run
run: ## Run the controller against the cluster in ~/.kube/config
	go run ./cmd

.PHONY: docker-build
docker-build: ## Build the controller image
	docker build -t $(IMG) .

.PHONY: install
install: ## Install the CRD into the cluster
	kubectl apply -f config/crd/bases/

.PHONY: uninstall
uninstall: ## Remove the CRD from the cluster
	kubectl delete -f config/crd/bases/

.PHONY: sample
sample: ## Apply the sample FailoverApp
	kubectl apply -f config/samples/
