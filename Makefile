GLIDE_VERSION = v0.13.1

# If no target is defined, assume the host is the target.
ifeq ($(origin GOOS), undefined)
	GOOS := $(shell go env GOOS)
endif
ifeq ($(origin GOARCH), undefined)
	GOARCH := $(shell go env GOARCH)
endif

# Set the build version
ifeq ($(origin VERSION), undefined)
	VERSION := $(shell git describe --tags --always --dirty)
endif

build: get-deps
	GOOS=linux go build -o bin/linux/provision ./provision
	GOOS=darwin go build -o bin/darwin/provision ./provision

tools/glide-$(GOOS)-$(GOARCH):
	mkdir -p tools
	curl -L https://github.com/Masterminds/glide/releases/download/$(GLIDE_VERSION)/glide-$(GLIDE_VERSION)-$(GOOS)-$(GOARCH).tar.gz | tar -xz -C tools
	mv tools/$(GOOS)-$(GOARCH)/glide tools/glide-$(GOOS)-$(GOARCH)
	rm -r tools/$(GOOS)-$(GOARCH)

get-deps: tools/glide-$(GOOS)-$(GOARCH)
	tools/glide-$(GOOS)-$(GOARCH) install -v

publish:
	@echo Triggering a build that should publish artifacts ot GitHub
	curl -u $(CIRCLE_CI_TOKEN): -X POST --header "Content-Type: application/json" \
		-d '{"tag": "$(VERSION)"}'                      \
		https://circleci.com/api/v1.1/project/github/apprenda/kismatic-provision