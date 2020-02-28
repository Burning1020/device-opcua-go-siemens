.PHONY: build test clean docker run build-arm64 docker-arm64

GO=CGO_ENABLED=0 GO111MODULE=on go

MICROSERVICES=cmd/device-opcua

.PHONY: $(MICROSERVICES) $(MICROSERVICES-arm64)

VERSION=$(shell cat ./VERSION)
GIT_SHA=$(shell git rev-parse HEAD)

GOFLAGS=-ldflags "-X github.com/edgexfoundry/device-opcua-go.Version=$(VERSION)"

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
		-t burning1020/docker-device-opcua-go:$(VERSION)-dev \
		.

run:
	cd bin && ./edgex-launch.sh
