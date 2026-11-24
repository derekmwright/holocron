.PHONY: test integration build
test:
	go test ./...
integration:
	bash scripts/test-integration.sh
build:
	go build -buildvcs=false ./cmd/holocron
