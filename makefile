

build:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/bifrost server/bifrost/main.go
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/testServer server/test_server/main.go