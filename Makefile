.PHONY: web dist build run tidy

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X goread/internal/version.Version=$(VERSION)

web:
	cd web && npm ci && npm run build

dist: web
	rm -rf internal/ui/dist
	mkdir -p internal/ui/dist
	cp -r web/dist/. internal/ui/dist/

build: dist
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o goread ./cmd/app

run: build
	./goread

tidy:
	go mod tidy
