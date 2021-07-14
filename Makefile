# Capture image tag from git branch name
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null || true)
ifeq (,$(GIT_BRANCH))
TAG = latest
else ifeq (master, $(GIT_BRANCH))
TAG = latest
else ifeq (HEAD, $(GIT_BRANCH))
TAG = $(shell git describe --abbrev=0 --tags $(shell git rev-list --abbrev-commit --tags --max-count=1) 2> /dev/null || true)
else
TAG = $(GIT_BRANCH)
endif

# Image URL to use all building/pushing image targets
IMG ?= netrisai/netris-operator:$(TAG)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./controllers/..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

release: generate fmt vet manifests kustomize
	$(KUSTOMIZE) build config/crd > deploy/netris-operator.crds.yaml
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > deploy/netris-operator.yaml

pip-install-reqs:
	pip3 install yq pyyaml

helm: generate fmt vet manifests pip-install-reqs
	mkdir -p deploy/charts/netris-operator/crds/
	cp config/crd/bases/* deploy/charts/netris-operator/crds/
	echo "{{- if .Values.rbac.create -}}" > deploy/charts/netris-operator/templates/rbac.yaml
	for i in $(shell yq -y .resources config/rbac/kustomization.yaml | awk {'print $$2'});\
	do echo "---" >> deploy/charts/netris-operator/templates/rbac.yaml && \
	scripts/rbac-helm-template.py config/rbac/$${i} | yq -y . >> deploy/charts/netris-operator/templates/rbac.yaml;\
	done
	echo "{{- end }}" >> deploy/charts/netris-operator/templates/rbac.yaml

helm-push: helm
	@{ \
	set -e ;\
	HELM_CHART_GEN_TMP_DIR=$$(mktemp -d) ;\
	git clone git@github.com:netrisai/charts.git --depth 1 $$HELM_CHART_GEN_TMP_DIR ;\
	if [[ -z "$${HELM_CHART_REPO_COMMIT_MSG}" ]]; then HELM_CHART_REPO_COMMIT_MSG=Update-$$(date '+%F_%T' -u); fi ;\
	cp -r deploy/charts $$HELM_CHART_GEN_TMP_DIR ;\
	cd $$HELM_CHART_GEN_TMP_DIR ;\
	git add charts && git commit -m $$HELM_CHART_REPO_COMMIT_MSG && git push -u origin main ;\
	rm -rf $$HELM_CHART_GEN_TMP_DIR ;\
	}
