.PHONY: dev build cli check fmt

dev:
	go tool mygo dev

build:
	go tool mygo build

cli:
	go build -o build/sento ./cmd/sento

check:
	go test ./...
	go vet ./...
	go tool mygo vet .

fmt:
	gofmt -w *.go internal cmd
