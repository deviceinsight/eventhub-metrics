BUILD_TS := $(shell date -Iseconds -u)
COMMIT_SHA ?= $(shell git rev-parse HEAD)
VERSION ?= $(shell git diff --quiet && git describe --exact-match --tags $(COMMIT_SHA) 2>/dev/null || echo "latest")
DOCKER_REGISTRY ?= docker.io
DOCKER_GROUP ?= deviceinsight

export CGO_ENABLED=0

ld_flags := "-X main.Version=$(VERSION) -X main.GitCommit=$(COMMIT_SHA) -X main.BuildTime=$(BUILD_TS)"

FILES    := $(shell find . -name '*.go' -type f -not -name '*.pb.go' -not -name '*_generated.go' -not -name '*_test.go')
TESTS    := $(shell find . -name '*.go' -type f -not -name '*.pb.go' -not -name '*_generated.go' -name '*_test.go')

.DEFAULT_GOAL := all

.PHONY: all
all: lint build # Runs all code checks

.PHONY: fmt
fmt: # Auto-format source files
	go mod tidy
	gofmt -s -l -w $(FILES) $(TESTS)
	goimports -l -w $(FILES) $(TESTS)

.PHONY: update-dependencies
update-dependencies: # update dependencies to latest MINOR.PATCH
	go get -t -u ./...

.PHONY: cve-check
cve-check: generate # Check for vulnerabilities
	govulncheck ./...

.PHONY: lint
lint: # Runs linter
	golangci-lint run

.PHONY: build
build: # Creates a release build
	go build -ldflags $(ld_flags) cmd/app/eventhub_metrics.go

.PHONY: install
install: # Installs a release build
	go install -ldflags $(ld_flags) cmd/app/eventhub_metrics.go

.PHONY: docker-build
docker-build: # Creates a docker image
	docker build -t eventhub-metrics:$(VERSION) --build-arg COMMIT_SHA=$(COMMIT_SHA) --build-arg VERSION=$(VERSION) --platform linux/amd64 .

.PHONY: docker-push
docker-push: # Pushes the docker images to the registry
	docker tag eventhub-metrics:$(VERSION) $(DOCKER_REGISTRY)/$(DOCKER_GROUP)/eventhub-metrics:$(VERSION)
	docker tag eventhub-metrics:$(VERSION) $(DOCKER_REGISTRY)/$(DOCKER_GROUP)/eventhub-metrics:latest
	docker push $(DOCKER_REGISTRY)/$(DOCKER_GROUP)/eventhub-metrics:$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_GROUP)/eventhub-metrics:latest

.PHONY: docker-publish
docker-publish: docker-build docker-push # build and push docker image

.PHONY: helm-package
helm-package: # package helm chart
	helm package ./helm/eventhub-metrics --version $(VERSION)
	mv eventhub-metrics-$(VERSION).tgz ./helm/archives
	helm repo index ./helm/archives

# usage make release version=2.5.0
#
.PHONY: release
release: docker-publish helm-package
	git add helm/archives/.
	git commit -m "releases $(VERSION)"
	git tag -a v$(VERSION) -m "release v$(VERSION)"
	git push origin
	git push origin v$(VERSION)
