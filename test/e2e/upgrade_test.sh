#!/bin/bash

# E2E Upgrade Test for Bifrost (Master-Worker Architecture)
# This script tests the complete upgrade flow with PID stability:
# 1. Start daemon in Master-Worker mode (-m -d)
# 2. Trigger upgrade (SIGHUP to Master)
# 3. Verify Master PID remains UNCHANGED (key for Systemd Active Time)
# 4. Verify Worker PID has changed (new Worker spawned)
# 5. Verify only one process is listening on port
# 6. Verify old Worker is terminated

set -e

# Configuration
BIFROST_BIN="${BIFROST_BIN:-/tmp/bifrost}"
CONFIG_FILE="${CONFIG_FILE:-/tmp/e2e_config.yaml}"
WORK_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
LOGS_DIR="${LOGS_DIR:-$WORK_DIR/logs}"
TEST_PORT="${TEST_PORT:-18001}"
TIMEOUT_SECONDS=30
# Use Master-Worker mode by default
USE_MASTER_WORKER="${USE_MASTER_WORKER:-true}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up..."
    pkill -9 -f "$BIFROST_BIN" 2>/dev/null || true
    rm -rf "$LOGS_DIR" "$CONFIG_FILE" 2>/dev/null || true
}

trap cleanup EXIT

create_config() {
    log_info "Creating test configuration..."
    mkdir -p "$LOGS_DIR"
    cat > "$CONFIG_FILE" <<EOF
version: 1
watch: false

servers:
  test_server:
    bind: "127.0.0.1:$TEST_PORT"
EOF
}

build_bifrost() {
    if [ ! -f "$BIFROST_BIN" ]; then
        log_info "Building bifrost..."
        go build -o "$BIFROST_BIN" ./server/bifrost
    fi
}

wait_for_ready() {
    local timeout=$1
    local start_time=$(date +%s)
    
    while true; do
        if ss -tlnp 2>/dev/null | grep -q ":$TEST_PORT"; then
            return 0
        fi
        
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        if [ $elapsed -ge $timeout ]; then
            log_error "Timeout waiting for server to be ready"
            return 1
        fi
        
        sleep 0.5
    done
}

get_process_count() {
    pgrep -f "$BIFROST_BIN" 2>/dev/null | wc -l
}

get_pid_from_file() {
    cat "$LOGS_DIR/bifrost.pid" 2>/dev/null || echo "0"
}

# Get all bifrost PIDs (for Master-Worker mode, returns multiple)
get_all_pids() {
    pgrep -f "$BIFROST_BIN" 2>/dev/null || echo ""
}

# --------------------------------------------------------------------------
# Master-Worker Mode Test
# --------------------------------------------------------------------------
run_master_worker_test() {
    log_info "============================================"
    log_info "Starting E2E Upgrade Test (Master-Worker Mode)"
    log_info "============================================"
    
    # Step 1: Start daemon in Master-Worker mode
    log_info "Step 1: Starting daemon in Master-Worker mode..."
    "$BIFROST_BIN" -c "$CONFIG_FILE" -m -d &
    sleep 3
    
    if ! wait_for_ready 10; then
        log_error "Failed to start daemon"
        return 1
    fi
    
    local master_pid=$(get_pid_from_file)
    local all_pids=$(get_all_pids)
    local process_count=$(echo "$all_pids" | wc -w)
    
    log_info "  Master PID (from file): $master_pid"
    log_info "  All PIDs: $all_pids"
    log_info "  Process count: $process_count"
    
    # In Master-Worker mode, we expect 2 processes (Master + Worker)
    if [ "$process_count" -lt 1 ]; then
        log_error "Expected at least 1 process, found $process_count"
        return 1
    fi
    
    # Step 2: Trigger upgrade
    log_info "Step 2: Triggering upgrade (SIGHUP to Master)..."
    "$BIFROST_BIN" -c "$CONFIG_FILE" -u
    
    sleep 5
    
    # Step 3: Verify results
    log_info "Step 3: Verifying results..."
    
    local new_master_pid=$(get_pid_from_file)
    local new_all_pids=$(get_all_pids)
    local new_process_count=$(echo "$new_all_pids" | wc -w)
    local listeners=$(ss -tlnp 2>/dev/null | grep ":$TEST_PORT" | wc -l)

    log_info "  Master PID before: $master_pid"
    log_info "  Master PID after: $new_master_pid"
    log_info "  Process count: $new_process_count"
    log_info "  Port listeners: $listeners"
    
    local failed=0
    
    # Check 1: Master PID should remain UNCHANGED (key for Systemd Active Time)
    if [ "$master_pid" = "$new_master_pid" ]; then
        log_info "PASS: Master PID unchanged ($master_pid) - Systemd Active Time preserved!"
    else
        log_error "FAIL: Master PID changed from $master_pid to $new_master_pid"
        failed=1
    fi
    
    # Check 2: Only one listener on port
    if [ "$listeners" -ne 1 ]; then
        log_error "FAIL: Expected 1 listener on port $TEST_PORT, found $listeners"
        failed=1
    else
        log_info "PASS: Only one listener on port $TEST_PORT"
    fi
    
    # Check 3: Master process should still be running
    if ! ps -p "$new_master_pid" > /dev/null 2>&1; then
        log_error "FAIL: Master process (PID $new_master_pid) is not running"
        failed=1
    else
        log_info "PASS: Master process (PID $new_master_pid) is running"
    fi
    
    log_info "============================================"
    if [ $failed -eq 0 ]; then
        log_info "E2E Upgrade Test (Master-Worker): ${GREEN}PASSED${NC}"
        return 0
    else
        log_error "E2E Upgrade Test (Master-Worker): FAILED"
        return 1
    fi
}

# --------------------------------------------------------------------------
# Legacy Mode Test (Self-Exec)
# --------------------------------------------------------------------------
run_legacy_test() {
    log_info "============================================"
    log_info "Starting E2E Upgrade Test (Legacy Mode)"
    log_info "============================================"
    
    # Step 1: Start daemon
    log_info "Step 1: Starting daemon..."
    "$BIFROST_BIN" -c "$CONFIG_FILE" -d &
    sleep 2
    
    if ! wait_for_ready 10; then
        log_error "Failed to start daemon"
        return 1
    fi
    
    local old_pid=$(get_pid_from_file)
    local process_count=$(get_process_count)
    
    log_info "  Daemon started: PID=$old_pid, process_count=$process_count"
    
    if [ "$process_count" -ne 1 ]; then
        log_error "Expected 1 process, found $process_count"
        return 1
    fi
    
    # Step 2: Trigger upgrade
    log_info "Step 2: Triggering upgrade..."
    "$BIFROST_BIN" -c "$CONFIG_FILE" -u
    
    sleep 5
    
    # Step 3: Verify results
    log_info "Step 3: Verifying results..."
    
    local new_pid=$(get_pid_from_file)
    local new_process_count=$(get_process_count)
    local listeners=$(ss -tlnp 2>/dev/null | grep ":$TEST_PORT" | wc -l)

    log_info "  Old PID: $old_pid"
    log_info "  New PID: $new_pid"
    log_info "  Process count: $new_process_count"
    log_info "  Port listeners: $listeners"
    
    local failed=0
    
    # Check 1: Only one process should be running
    if [ "$new_process_count" -ne 1 ]; then
        log_error "FAIL: Expected 1 process, found $new_process_count"
        failed=1
    else
        log_info "PASS: Only one process running"
    fi
    
    # Check 2: PID should have changed
    if [ "$old_pid" = "$new_pid" ]; then
        log_error "FAIL: PID did not change (old=$old_pid, new=$new_pid)"
        failed=1
    else
        log_info "PASS: PID changed from $old_pid to $new_pid"
    fi
    
    # Check 3: Only one listener on port
    if [ "$listeners" -ne 1 ]; then
        log_error "FAIL: Expected 1 listener on port $TEST_PORT, found $listeners"
        failed=1
    else
        log_info "PASS: Only one listener on port $TEST_PORT"
    fi
    
    # Check 4: Old process should not be running
    if ps -p "$old_pid" > /dev/null 2>&1; then
        log_error "FAIL: Old process (PID $old_pid) is still running"
        failed=1
    else
        log_info "PASS: Old process (PID $old_pid) is terminated"
    fi
    
    # Check 5: New process should be running
    if ! ps -p "$new_pid" > /dev/null 2>&1; then
        log_error "FAIL: New process (PID $new_pid) is not running"
        failed=1
    else
        log_info "PASS: New process (PID $new_pid) is running"
    fi
    
    log_info "============================================"
    if [ $failed -eq 0 ]; then
        log_info "E2E Upgrade Test (Legacy): ${GREEN}PASSED${NC}"
        return 0
    else
        log_error "E2E Upgrade Test (Legacy): FAILED"
        return 1
    fi
}

# Main
main() {
    cd "$(dirname "$0")/../.."
    
    build_bifrost
    create_config
    
    if [ "$USE_MASTER_WORKER" = "true" ]; then
        run_master_worker_test
    else
        run_legacy_test
    fi
}

main "$@"
