

FULL_COMMIT := $(shell git rev-parse HEAD)
LDFLAGS = -ldflags "-X main.Build=$(FULL_COMMIT)"


.PHONY: test
test:
	go clean -testcache
	go test -race -coverprofile=cover.out -covermode=atomic ./pkg/... -v

lint:
	golangci-lint run --timeout 5m --verbose ./pkg/... -v

docker_lint:
	docker run -it --rm -v "${LOCAL_WORKSPACE_FOLDER}:/app" -w /app golangci/golangci-lint:v1.62.0-alpine golangci-lint run --timeout 5m --verbose ./pkg/...


build:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build $(LDFLAGS) -o bin/bifrost server/bifrost/main.go
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/testServer server/testserver/main.go

release: build lint test