#!/bin/bash

# E2E Upgrade Test for Bifrost (Master-Worker Architecture)
# This script tests the complete hot reload flow with PID stability:
# 1. Start Master in foreground mode (background via &)
# 2. Trigger upgrade via SIGHUP
# 3. Verify Master PID remains UNCHANGED
# 4. Verify Worker PID has changed (new Worker spawned)
# 5. Verify only one process is listening on port
# 6. Test KeepAlive (Worker crash recovery)

set -e

# Configuration
WORK_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
BIFROST_BIN="${BIFROST_BIN:-$WORK_DIR/bin/bifrost}"
CONFIG_FILE="${CONFIG_FILE:-$WORK_DIR/test/e2e/config.yaml}"
LOGS_DIR="${LOGS_DIR:-$WORK_DIR/logs}"
TEST_PORT="${TEST_PORT:-18001}"
TIMEOUT_SECONDS=30

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
}

trap cleanup EXIT

build_bifrost() {
    if command -v go >/dev/null 2>&1; then
        log_info "Building bifrost..."
        go build -o "$BIFROST_BIN" ./server/bifrost
    else
        log_warn "go command not found, skipping build and using existing binary (if any)."
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

# Get all bifrost PIDs (Master + Workers)
get_all_pids() {
    pgrep -f "$BIFROST_BIN" 2>/dev/null || echo ""
}

# Get the Master PID (parent of all Worker PIDs)
get_master_pid() {
    # The first PID returned by pgrep is usually the Master
    pgrep -f "$BIFROST_BIN" 2>/dev/null | head -n 1
}

# --------------------------------------------------------------------------
# Master-Worker Mode Test
# --------------------------------------------------------------------------
run_master_worker_test() {
    log_info "============================================"
    log_info "Starting E2E Test (Master-Worker Architecture)"
    log_info "============================================"
    
    # Step 1: Start Master in foreground mode (background via &)
    log_info "Step 1: Starting Master in foreground mode..."
    "$BIFROST_BIN" -c "$CONFIG_FILE" 2>&1 | tee "$LOGS_DIR/master_debug.log" &
    local master_shell_pid=$!
    sleep 3
    
    if ! wait_for_ready 10; then
        log_error "Failed to start daemon"
        return 1
    fi
    
    # Get Master PID (actual bifrost process, not the tee pipe)
    local master_pid=$(get_master_pid)
    local initial_pids=$(get_all_pids)
    local initial_count=$(echo "$initial_pids" | wc -w)
    
    log_info "  Master PID: $master_pid"
    log_info "  Initial PIDs: $initial_pids"
    
    if [ "$initial_count" -lt 2 ]; then
        log_error "Expected at least 2 processes (Master + Worker), found $initial_count"
        return 1
    fi

    # Identify worker PID (one of the PIDs that is NOT the master_pid)
    local initial_worker_pid=""
    for p in $initial_pids; do
        if [ "$p" != "$master_pid" ]; then
            initial_worker_pid=$p
            break
        fi
    done
    log_info "  Initial Worker PID: $initial_worker_pid"

    # ----------------------------------------------------------------------
    # Scenario 1: KeepAlive Test (Worker Crashed/Killed)
    # ----------------------------------------------------------------------
    log_info "Step 2: Scenario - KeepAlive Test (Killing Worker)..."
    kill -9 "$initial_worker_pid"
    log_info "  Sent SIGKILL to Worker ($initial_worker_pid)"
    
    sleep 5 # Wait for Master to detect and restart
    
    local post_kill_pids=$(get_all_pids)
    local new_worker_pid=""
    local current_master_pid=$(get_master_pid)
    for p in $post_kill_pids; do
        if [ "$p" != "$current_master_pid" ]; then
            new_worker_pid=$p
            break
        fi
    done
    
    log_info "  Master PID after Kill: $current_master_pid"
    log_info "  New Worker PID after Kill: $new_worker_pid"
    
    local failed=0
    if [ "$master_pid" != "$current_master_pid" ]; then
        log_error "FAIL: Master PID changed during KeepAlive test"
        failed=1
    else
        log_info "PASS: Master PID remains constant ($master_pid)"
    fi
    
    if [ -z "$new_worker_pid" ] || [ "$new_worker_pid" = "$initial_worker_pid" ]; then
        log_error "FAIL: Worker did not restart or PID stayed the same"
        failed=1
    else
        log_info "PASS: Worker restarted with new PID ($new_worker_pid)"
    fi

    # ----------------------------------------------------------------------
    # Scenario 2: Hot Reload Test (SIGHUP)
    # ----------------------------------------------------------------------
    log_info "Step 3: Scenario - Hot Reload Test (SIGHUP)..."
    
    # Send SIGHUP directly to Master PID (no -u flag)
    kill -HUP "$master_pid"
    log_info "  Sent SIGHUP to Master ($master_pid)"
    
    sleep 5
    
    local post_upgrade_master_pid=$(get_master_pid)
    local final_pids=$(get_all_pids)
    local final_worker_pid=""
    for p in $final_pids; do
        if [ "$p" != "$post_upgrade_master_pid" ]; then
            final_worker_pid=$p
            break
        fi
    done

    log_info "  Master PID after Upgrade: $post_upgrade_master_pid"
    log_info "  Worker PID after Upgrade: $final_worker_pid"

    if [ "$master_pid" != "$post_upgrade_master_pid" ]; then
        log_error "FAIL: Master PID changed during Hot Reload from $master_pid to $post_upgrade_master_pid"
        failed=1
    else
        log_info "PASS: Master PID remains constant ($master_pid)"
    fi

    if [ "$final_worker_pid" = "$new_worker_pid" ]; then
        log_error "FAIL: Worker PID did not change after upgrade"
        failed=1
    else
        log_info "PASS: New Worker spawned after upgrade ($final_worker_pid)"
    fi

    local listeners=$(ss -tlnp 2>/dev/null | grep ":$TEST_PORT" | wc -l)
    if [ "$listeners" -ne 1 ]; then
        log_error "FAIL: Expected 1 listener on port $TEST_PORT, found $listeners"
        failed=1
    else
        log_info "PASS: Exactly 1 port listener active"
    fi

    # ----------------------------------------------------------------------
    # Scenario 3: Graceful Shutdown (SIGTERM)
    # ----------------------------------------------------------------------
    log_info "Step 4: Scenario - Graceful Shutdown (SIGTERM)..."
    kill -TERM "$master_pid"
    log_info "  Sent SIGTERM to Master ($master_pid)"
    
    sleep 3
    
    local post_shutdown_count=$(get_process_count)
    if [ "$post_shutdown_count" -ne 0 ]; then
        log_error "FAIL: Expected 0 processes after shutdown, found $post_shutdown_count"
        failed=1
    else
        log_info "PASS: All processes terminated after SIGTERM"
    fi

    log_info "============================================"
    if [ $failed -eq 0 ]; then
        log_info "Master-Worker E2E Tests: ${GREEN}PASSED${NC}"
        return 0
    else
        log_error "Master-Worker E2E Tests: FAILED"
        return 1
    fi
}

# Main
main() {
    cd "$(dirname "$0")/../.."
    
    build_bifrost
    mkdir -p "$LOGS_DIR"
    
    run_master_worker_test
}

main "$@"
