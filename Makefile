.PHONY: web dist build run tidy

web:
	cd web && npm ci && npm run build

dist: web
	rm -rf internal/ui/dist
	mkdir -p internal/ui/dist
	cp -r web/dist/. internal/ui/dist/

build: dist
	CGO_ENABLED=0 go build -o ebook-reader ./cmd/app

run: build
	./ebook-reader

tidy:
	go mod tidy
