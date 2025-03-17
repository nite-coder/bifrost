.PHONY: test
test:
	go clean -testcache
	go test -race -coverprofile=cover.out -covermode=atomic ./pkg/... -v

lint:
	golangci-lint cache clean
	golangci-lint run --timeout 5m --verbose ./pkg/... -v

docker.lint:
	docker run -it --rm -v "${LOCAL_WORKSPACE_FOLDER}:/app" -w /app golangci/golangci-lint:v1.63.4-alpine golangci-lint run --timeout 5m --verbose ./pkg/...


build:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/bifrost server/bifrost/main.go
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/testServer server/testserver/main.go

release: build lint test


coverage:
	go tool cover -func=cover.out