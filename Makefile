.PHONY: build test clean docker run build-arm64 docker-arm64

GO=CGO_ENABLED=0 GO111MODULE=on go
GO_arm64=CGO_ENABLED=0 GO111MODULE=on GOARCH=arm64 go

MICROSERVICES=cmd/device-opcua
MICROSERVICES_arm64=cmd/device-opcua-arm64

.PHONY: $(MICROSERVICES) $(MICROSERVICES-arm64)

VERSION=$(shell cat ./VERSION)

GOFLAGS=-ldflags "-X github.com/edgexfoundry/device-opcua-go.Version=$(VERSION)"
GOFLAGS_arm64=-ldflags "-X github.com/edgexfoundry/device-opcua-go-arm64.Version=$(VERSION)"

GIT_SHA=$(shell git rev-parse HEAD)

build: $(MICROSERVICES)
	$(GO) build ./...

cmd/device-opcua:
	$(GO) build $(GOFLAGS) -o $@ ./cmd

test:
	go test ./... -cover

clean:
	rm -f $(MICROSERVICES)

docker:
	docker build \
		--label "git_sha=$(GIT_SHA)" \
		-t edgexfoundry/docker-device-opcua-go:$(GIT_SHA) \
		-t edgexfoundry/docker-device-opcua-go:$(VERSION)-dev \
		.

run:
	cd bin && ./edgex-launch.sh

build-arm64: $(MICROSERVICES_arm64)
	$(GO_arm64) build ./...

cmd/device-opcua-arm64:
	$(GO_arm64) build $(GOFLAGS_arm64) -o $@ ./cmd
	
docker-arm64:
	docker build \
		-f Dockerfile_ARM64 \
		--label "git_sha=$(GIT_SHA)" \
		-t 192.168.3.128:5000/docker-device-opcua-go-arm64 \
		.
