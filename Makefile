.PHONY: build test clean prepare update

GO=CGO_ENABLED=0 GO111MODULE=on go

MICROSERVICES=cmd/device-opcua
.PHONY: $(MICROSERVICES)

VERSION=$(shell cat ./VERSION)

GOFLAGS=-ldflags "-X github.com/edgexfoundry/device-opcua-go.Version=$(VERSION)"

build: $(MICROSERVICES)
	$(GO) build ./...

cmd/device-opcua:
	$(GO) build $(GOFLAGS) -o $@ ./cmd

test:
	go test ./... -cover

clean:
	rm -f $(MICROSERVICES)

update:
	glide update
	
run:
	cd bin && ./edgex-launch.sh
