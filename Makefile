VERSION    := 0.2.1
USERNAME   := unblee
BINNAME    := mirror
REPONAME   := $(BINNAME)
REVISION   := $(shell git rev-parse --short HEAD)
GO_VERSION := 1.8.0
LDFLAGS    := -w -s \
              -X main.Version=$(VERSION) \
              -X main.Revision=$(REVISION) \
              -X main.GoVersion=$(GO_VERSION) \
              -extldflags '-static'

BUILD_REPO_ROOT     := /go/src/$(REPONAME)
RELEASE_BIN_DIR     := $(BUILD_REPO_ROOT)/release/bin
RELEASE_ARCHIVE_DIR := $(BUILD_REPO_ROOT)/release/archive

DOCKER_ENV := -e VERSION=$(VERSION) \
              -e USERNAME=$(USERNAME) \
              -e BINNAME=$(BINNAME) \
              -e REPONAME=$(REPONAME) \
              -e LDFLAGS='$(LDFLAGS)' \
              -e BUILD_REPO_ROOT='$(BUILD_REPO_ROOT)' \
              -e RELEASE_BIN_DIR='$(RELEASE_BIN_DIR)' \
              -e RELEASE_ARCHIVE_DIR='$(RELEASE_ARCHIVE_DIR)' \
              -e GITHUB_TOKEN='$(GITHUB_TOKEN)'

DOCKER_RUN_OPT := -v $(PWD):$(BUILD_REPO_ROOT) \
                  -w /go/src/$(BINNAME) \
                  $(DOCKER_ENV)
DOCKER_RUN     := docker run -it --rm $(DOCKER_RUN_OPT) golang:$(GO_VERSION)-alpine

.PHONY: gh-release
gh-release: archive
	git push
	$(DOCKER_RUN) sh scripts/gh_release.sh

.PHONY: archive
archive: bin-build
	$(DOCKER_RUN) sh scripts/archive.sh

.PHONY: dh-release
dh-release: docker-build
	docker push $(USERNAME)/$(BINNAME):$(VERSION)
	docker push $(USERNAME)/$(BINNAME):latest

.PHONY: docker-build
docker-build: bin-build
	cp -f release/bin/$(BINNAME)-$(VERSION)-linux-amd64/$(BINNAME) $(BINNAME)
	docker build -t $(USERNAME)/$(BINNAME):$(VERSION) .
	docker tag $(USERNAME)/$(BINNAME):$(VERSION) $(USERNAME)/$(BINNAME):latest
	rm -f $(BINNAME)

.PHONY: bin-build
bin-build: deps
	$(DOCKER_RUN) sh scripts/build.sh

.PHONY: deps
deps:
	glide install

.PHONY: example-up
example-up: example-stop docker-build
	{ \
		cd example; \
		docker-compose up -d; \
	}

.PHONY: example-stop
example-stop:
	{ \
		cd example; \
		docker-compose stop; \
		docker-compose rm -f; \
	}

.PHONY: test
test:
	go test -v `glide novendor | grep -v example`

.PHONY: clean
clean:
	rm -fr release/archive
	rm -fr release/bin