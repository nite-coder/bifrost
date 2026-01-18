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

# Get the Master PID
get_master_pid() {
    ps -e -o pid=,comm= | grep 'bifrost-master' | awk '{print $1}' | head -n 1
}

# Get Worker PIDs
get_worker_pids() {
    ps -e -o pid=,comm= | grep 'bifrost-worker' | awk '{print $1}'
}

# Get all bifrost PIDs (Master + Workers)
get_all_pids() {
    get_master_pid
    get_worker_pids
}

get_process_count() {
    get_all_pids | grep -v '^$' | wc -l
}

# --------------------------------------------------------------------------
# Master-Worker Mode Test
# --------------------------------------------------------------------------
run_master_worker_test() {
    log_info "============================================"
    log_info "Starting E2E Test (Master-Worker Architecture)"
    log_info "============================================"
    
    # Step 1: Start Master in foreground mode
    log_info "Step 1: Starting Master..."
    rm -f "$LOGS_DIR/e2e_test.log"
    
    "$BIFROST_BIN" -c "$CONFIG_FILE" > "$LOGS_DIR/master_stdout.log" 2>&1 &
    local master_shell_pid=$!
    sleep 3
    
    if ! wait_for_ready 15; then
        log_error "Failed to start bifrost"
        cat "$LOGS_DIR/master_stdout.log"
        return 1
    fi
    
    # Get Master PID and Worker PID
    local master_pid=$(get_master_pid)
    local worker_pids=$(get_worker_pids)
    local initial_count=$(get_process_count)
    
    log_info "  Master PID: $master_pid"
    log_info "  Worker PIDs: $worker_pids"
    
    if [ -z "$master_pid" ]; then
        log_error "FAIL: Master process (bifrost-master) not found"
        return 1
    fi
    log_info "PASS: Master process name is correctly set to 'bifrost-master'"

    if [ -z "$worker_pids" ]; then
        log_error "FAIL: Worker process (bifrost-worker) not found"
        return 1
    fi
    log_info "PASS: Worker process name is correctly set to 'bifrost-worker'"

    local initial_worker_pid=$(echo "$worker_pids" | head -n 1)

    # ----------------------------------------------------------------------
    # Scenario 1: KeepAlive Test (Worker Crashed/Killed)
    # ----------------------------------------------------------------------
    log_info "Step 2: Scenario - KeepAlive Test (Killing Worker)..."
    kill -9 "$initial_worker_pid"
    log_info "  Sent SIGKILL to Worker ($initial_worker_pid)"
    
    sleep 5 # Wait for Master to detect and restart
    
    local current_master_pid=$(get_master_pid)
    local new_worker_pids=$(get_worker_pids)
    local new_worker_pid=$(echo "$new_worker_pids" | head -n 1)
    
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
    
    # Send SIGHUP to Master
    kill -HUP "$master_pid"
    log_info "  Sent SIGHUP to Master ($master_pid)"
    
    sleep 8 # Wait for reload to complete (including wait for FDs)
    
    local post_upgrade_master_pid=$(get_master_pid)
    local final_worker_pids=$(get_worker_pids)
    local final_worker_pid=$(echo "$final_worker_pids" | tail -n 1) # Get the newest one

    log_info "  Master PID after Upgrade: $post_upgrade_master_pid"
    log_info "  Worker PID after Upgrade: $final_worker_pid"

    if [ "$master_pid" != "$post_upgrade_master_pid" ]; then
        log_error "FAIL: Master PID changed during Hot Reload"
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

    # ----------------------------------------------------------------------
    # Scenario 3: Log Rotation Test (SIGUSR1)
    # ----------------------------------------------------------------------
    log_info "Step 4: Scenario - Log Rotation Test (SIGUSR1)..."
    
    local log_file="$LOGS_DIR/e2e_test.log"
    if [ ! -f "$log_file" ]; then
        log_error "FAIL: Log file $log_file not found"
        failed=1
    else
        local inode_before=$(stat -c %i "$log_file")
        log_info "  Log inode before rotation: $inode_before"
        
        # Simulate logrotate: move file away
        mv "$log_file" "$log_file.old"
        
        # Send SIGUSR1 to Master
        kill -USR1 "$master_pid"
        log_info "  Sent SIGUSR1 to Master ($master_pid)"
        
        sleep 2
        
        if [ ! -f "$log_file" ]; then
            log_error "FAIL: Log file not recreated after SIGUSR1"
            failed=1
        else
            local inode_after=$(stat -c %i "$log_file")
            log_info "  Log inode after rotation: $inode_after"
            
            if [ "$inode_before" = "$inode_after" ]; then
                log_error "FAIL: Log inode did not change after rotation"
                failed=1
            else
                log_info "PASS: Log file rotated and recreated with new inode"
            fi
        fi
    fi

    # ----------------------------------------------------------------------
    # Scenario 4: Graceful Shutdown (SIGTERM)
    # ----------------------------------------------------------------------
    log_info "Step 5: Scenario - Graceful Shutdown (SIGTERM)..."
    kill -TERM "$master_pid"
    log_info "  Sent SIGTERM to Master ($master_pid)"
    
    sleep 5
    
    local post_shutdown_count=$(get_process_count)
    if [ "$post_shutdown_count" -ne 0 ]; then
        log_error "FAIL: Expected 0 processes after shutdown, found $post_shutdown_count"
        get_all_pids
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
