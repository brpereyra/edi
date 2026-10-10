.PHONY: build test clean cross-compile install

BINARY_NAME=edi

build:
	go build -o $(BINARY_NAME) ./cmd/edi

test:
	go test -v ./...

install:
	go install ./cmd/edi

cross-compile:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/edi
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/edi
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/edi
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/edi
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/edi

clean:
	rm -f $(BINARY_NAME)
	rm -rf dist
