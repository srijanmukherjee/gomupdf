.PHONY: all fmt vet build test cover tidy clean

all: fmt vet test

# Format all packages.
fmt:
	gofmt -w .

# Report files that are not gofmt-clean (non-zero exit if any).
fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

# Static analysis.
vet:
	go vet ./...

# Compile every package (cgo; requires libmupdf installed).
build:
	go build ./...

# Run the test suite.
test:
	go test ./...

# Run tests with a coverage profile.
cover:
	go test -coverprofile=coverage.txt ./...
	go tool cover -html=coverage.txt -o coverage.html

tidy:
	go mod tidy

clean:
	go clean ./...
	rm -f coverage.txt coverage.html
