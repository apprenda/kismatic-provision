# Set the build version
ifeq ($(origin VERSION), undefined)
	VERSION := $(shell git describe --tags --always --dirty)
endif

build: get-deps
	GOOS=linux go build -o bin/linux/provision ./provision
	GOOS=darwin go build -o bin/darwin/provision ./provision

get-deps:
	go get github.com/onsi/ginkgo/ginkgo
	go get github.com/onsi/gomega
	go get github.com/jmcvetta/guid
	go get gopkg.in/yaml.v2
	go get -u github.com/aws/aws-sdk-go
	go get github.com/mitchellh/go-homedir
	go install github.com/onsi/ginkgo/ginkgo
	go get golang.org/x/crypto/ssh
	go get github.com/cloudflare/cfssl/csr
	go get github.com/packethost/packngo
	go get github.com/michaelbironneau/garbler/lib
	go get github.com/spf13/cobra
	go get golang.org/x/oauth2
	go get github.com/digitalocean/godo

publish:
	@echo Triggering a build that should publish artifacts ot GitHub
	curl -u $(CIRCLE_CI_TOKEN): -X POST --header "Content-Type: application/json" \
		-d '{"tag": "$(VERSION)"}'                      \
		https://circleci.com/api/v1.1/project/github/apprenda/kismatic-provision