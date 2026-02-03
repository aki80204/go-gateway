BINARY_NAME=bootstrap
ZIP_NAME=go-gateway.zip

.PHONY: build zip clean test lint

all: lint test build zip

build:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) main.go

zip:
	@echo "Creating deployment package ($(ZIP_NAME))..."
	@zip $(ZIP_NAME) $(BINARY_NAME)

clean:
	@if [ -f $(BINARY_NAME) ]; then rm $(BINARY_NAME); fi
	@if [ -f $(ZIP_NAME) ]; then rm $(ZIP_NAME); fi

test:
	go test -v ./...
	
lint:
	@golangci-lint run ./...