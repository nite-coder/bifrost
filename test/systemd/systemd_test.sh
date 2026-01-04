#!/bin/bash
# Systemd Integration Test Runner
# This script builds and runs the systemd test container

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONTAINER_NAME="bifrost-systemd-test"
IMAGE_NAME="bifrost-systemd-test:latest"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

log_info() { echo -e "[INFO] $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

cleanup() {
    log_info "Cleaning up..."
    docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
}

trap cleanup EXIT

# Build bifrost binary
log_info "Building bifrost binary..."
cd "$PROJECT_ROOT"
mkdir -p bin
go build -o bin/bifrost ./server/bifrost

# Copy binary to test directory (avoid .dockerignore issues)
cp bin/bifrost test/systemd/bifrost

# Build Docker image from test/systemd directory
log_info "Building systemd test Docker image..."
cd "$PROJECT_ROOT/test/systemd"
docker build -t "$IMAGE_NAME" .

# Run container with systemd
log_info "Starting systemd container..."

# Check if running in CI environment or privileged mode is available
if docker run --rm --privileged alpine echo "privileged" 2>/dev/null; then
    DOCKER_OPTS="--privileged -v /sys/fs/cgroup:/sys/fs/cgroup:rw"
else
    log_error "Docker privileged mode not available. Systemd tests require --privileged flag."
    exit 1
fi

docker run -d --name "$CONTAINER_NAME" \
    $DOCKER_OPTS \
    --cgroupns=host \
    "$IMAGE_NAME"

# Wait for container to start
log_info "Waiting for container to be ready..."
sleep 5

# Run tests inside container
log_info "Running systemd integration tests..."
if docker exec "$CONTAINER_NAME" /app/run_test.sh; then
    echo ""
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}Systemd Integration Test: PASSED${NC}"
    echo -e "${GREEN}============================================${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}============================================${NC}"
    echo -e "${RED}Systemd Integration Test: FAILED${NC}"
    echo -e "${RED}============================================${NC}"
    
    log_info "Container logs:"
    docker logs "$CONTAINER_NAME"
    
    log_info "Journal logs:"
    docker exec "$CONTAINER_NAME" journalctl -u bifrost --no-pager -n 100 2>/dev/null || true
    
    exit 1
fi
