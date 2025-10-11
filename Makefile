TAG?=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo "v0.0.0" )
COMMIT?=$(shell git rev-parse HEAD)

default: build

.PHONY: generate
generate:
	@go generate ./...

.PHONY: build
build:
	@COMMIT=$(COMMIT) TAG=$(TAG) ./scripts/build.sh

.PHONY: test
test:
	CGO_ENABLED=0 go test -cover --count=1 ./...

.PHONY: clean
clean:
	./scripts/clean.sh

.PHONY: verify
verify:
	./scripts/verify.sh

.PHONY: chart
chart:
	TAG=$(TAG) ./scripts/chart.sh

.PHONY: image
image:
	TAG=$(TAG) ./scripts/image.sh

.PHONY: help
help:
	@echo "Usage:"
	@echo "	make build		local build binary files"
	@echo "	make test		run unit tests"
	@echo "	make generate		run code generator"
	@echo "	make verify		verify modules"
	@echo "	make chart		package helm charts"
	@echo "	make image		local build container images"
	@echo "	make clean		clean up built files"
	@echo "	make help		show this message"
