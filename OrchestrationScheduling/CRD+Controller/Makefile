REGISTRY ?= docker.io/sangshen

VERSION ?= dev

IMG ?= box-controller:$(VERSION)

.PHONY: build-image
build-image:
	docker build -f Dockerfile -t $(REGISTRY)/$(IMG) .

.PHONY: push
push: build-image
	docker push $(REGISTRY)/$(IMG)

CONTROLLER_GEN = ${GOBIN}/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.15.0)

.PHONY: generate
generate: controller-gen
	controller-gen schemapatch:manifests=./crd,maxDescLen=0,generateEmbeddedObjectMeta=true output:dir=./crd paths=./pkg/apis/boxstatefulset/...