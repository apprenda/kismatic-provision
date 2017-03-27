# Set the build version
ifeq ($(origin VERSION), undefined)
	VERSION := $(shell git rev-parse --short HEAD)
endif

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

build: get-deps
	GOOS=linux go build -o bin/linux/provision ./provision
	GOOS=darwin go build -o bin/darwin/provision ./provision

