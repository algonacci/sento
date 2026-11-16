.PHONY: dev build check fmt

dev:
	go tool mygo dev

build:
	go tool mygo build

check:
	go test ./...
	go vet ./...
	go tool mygo vet .

fmt:
	gofmt -w main.go app.go view.go theme.go
