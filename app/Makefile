.PHONY: run test build-local build-lambda clean

# Run locally for development on :8080
run:
	go run main.go

# Execute all unit, integration, and watertight tests
test:
	go test -v ./...

# Build local standalone executable binary
build-local:
	go build -o molda-ai main.go

# Build production package for AWS Lambda (Runtime: provided.al2023 on ARM64 Graviton)
build-lambda:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -ldflags="-s -w" -o bootstrap main.go
	zip moldaai-lambda.zip bootstrap
	rm bootstrap

clean:
	rm -f bootstrap moldaai-lambda.zip molda-ai
