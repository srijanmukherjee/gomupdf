.PHONY: all fmt vet build test race bench cover tidy clean

all: fmt vet test race

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

# Run the suite under the race detector (guards the v1.2 worker pool).
race:
	go test -race ./...

# Run benchmarks (e.g. serial vs concurrent rendering).
bench:
	go test -bench . -benchmem -run '^$$' ./...

# Run tests with a coverage profile.
cover:
	go test -coverprofile=coverage.txt ./...
	go tool cover -html=coverage.txt -o coverage.html

tidy:
	go mod tidy

clean:
	go clean ./...
	rm -f coverage.txt coverage.html
