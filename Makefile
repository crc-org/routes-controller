TAG ?= $(shell git describe --match=NeVeRmAtCh --always --abbrev=40 --dirty)

LDFLAGS = -ldflags '-s -w -extldflags "-static"'

.PHONY: build
build:
	GOOS=linux CGO_ENABLED=0 go build $(LDFLAGS) -o routes-controller .

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

.PHONY: lint
lint:
	golangci-lint run

.PHONY: image
image:
	docker build -t quay.io/crcont/routes-controller:$(TAG) .
