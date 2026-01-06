.PHONY: test
test:
	go clean -testcache
	go tool gotestsum --format testname -- -race -coverprofile=cover.out -covermode=atomic ./pkg/... ./internal/pkg/... -v

# E2E upgrade test requires root privileges for daemon mode
# Run with: sudo make e2e-test
e2e-test: build
	sudo bash ./test/e2e/upgrade_test.sh


coverage: test
	go tool cover -func=cover.out

lint:
	golangci-lint cache clean
	golangci-lint run --timeout 5m --verbose ./pkg/... ./internal/pkg/... -v

lintd:
	docker run -it --rm -v "${LOCAL_WORKSPACE_FOLDER}:/app" -w /app golangci/golangci-lint:v2.7.2-alpine golangci-lint run --timeout 5m --verbose ./pkg/...


build:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/bifrost server/bifrost/main.go
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/testServer server/testserver/main.go

buildd:
	docker buildx build --tag jasonsoft/bifrost .

rund:
	docker run -it --rm --name bifrost --net=host \
		-v "${LOCAL_WORKSPACE_FOLDER}/server/bifrost/config.yaml:/app/config.yaml" \
		-v "${LOCAL_WORKSPACE_FOLDER}/server/bifrost/conf:/app/conf" \
		jasonsoft/bifrost 

release: build lint test e2e-test

k8s_apply:
	kubectl apply -f ./config/k8s/bifrost_deployment.yaml -f ./config/k8s/echo_deployment.yaml

k8s_del:
	kubectl delete -f ./config/k8s/bifrost_deployment.yaml -f ./config/k8s/echo_deployment.yaml

k8s_create:
	k3d cluster create mycluster \
	--servers 1 \
	--agents 0 \
	--port 30080:30080@server:0

k8s_show_logs:
	kubectl logs -l app=bifrost --all-containers=true -f
