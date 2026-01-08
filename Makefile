# Image URL to use all building/pushing image targets
IMG ?= korp-operator:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate CRD manifests
	$(CONTROLLER_GEN) crd:crdVersions=v1 paths="./api/..." output:crd:dir=./config/crd

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

.PHONY: test
test: manifests generate fmt vet ## Run tests
	go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build korp-operator binary
	go build -o bin/korp-operator cmd/manager/main.go

.PHONY: build-cli
build-cli: ## Build korp CLI binary
	go build -o bin/korp cmd/korp/main.go

.PHONY: run
run: manifests generate fmt vet ## Run operator from your host
	go run cmd/manager/main.go

.PHONY: docker-build
docker-build: test ## Build docker image
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image
	docker push ${IMG}

##@ Helm

.PHONY: helm-lint
helm-lint: ## Lint Helm chart
	helm lint charts/korp

.PHONY: helm-template
helm-template: ## Template Helm chart
	helm template korp charts/korp --namespace korp-operator

.PHONY: helm-package
helm-package: ## Package Helm chart
	helm package charts/korp -d dist/

.PHONY: helm-install
helm-install: ## Install using Helm
	helm install korp charts/korp --namespace korp-operator --create-namespace

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall Helm release
	helm uninstall korp --namespace korp-operator

.PHONY: helm-upgrade
helm-upgrade: ## Upgrade Helm release
	helm upgrade korp charts/korp --namespace korp-operator

.PHONY: helm-publish
helm-publish: ## Publish Helm chart to GitHub Pages
	./scripts/publish-helm-chart.sh

##@ Deployment

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster
	kubectl apply -f config/crd/

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster
	kubectl delete -f config/crd/

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster
	kubectl create namespace korp-operator --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f config/rbac/
	kubectl apply -f config/manager/

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster
	kubectl delete -f config/manager/
	kubectl delete -f config/rbac/

##@ Build Dependencies

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@latest)

# go-get-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
