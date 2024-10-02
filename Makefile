
.PHONY: test
test:
	go test -race -coverprofile=cover.out -covermode=atomic ./pkg/... -v

lint:
	golangci-lint run ./pkg/... -v

docker_lint:
	docker run -it --rm -v "${LOCAL_WORKSPACE_FOLDER}:/app" -w /app golangci/golangci-lint:v1.59.1-alpine golangci-lint run ./... -v


build:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/bifrost server/bifrost/main.go
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/testServer server/testserver/main.go

release: lint test