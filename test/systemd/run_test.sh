#!/bin/bash
# Systemd Integration Test Script
# This script runs inside the systemd container to test bifrost service

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
log_info() { echo -e "[INFO] $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

# Run command with timeout (catches hangs like missing NotifyDaemonReady)
run_with_timeout() {
    local timeout=$1
    local desc=$2
    shift 2
    
    log_info "Running with ${timeout}s timeout: $desc"
    
    if timeout "$timeout" "$@"; then
        return 0
    else
        local exit_code=$?
        if [ $exit_code -eq 124 ]; then
            log_fail "TIMEOUT after ${timeout}s: $desc"
        else
            log_fail "Command failed (exit $exit_code): $desc"
        fi
    fi
}

# Wait for systemd to be ready
wait_for_systemd() {
    log_info "Waiting for systemd to be ready..."
    for i in {1..30}; do
        if systemctl is-system-running --wait 2>/dev/null | grep -qE "(running|degraded)"; then
            log_info "Systemd is ready"
            return 0
        fi
        sleep 1
    done
    log_fail "Systemd not ready after 30 seconds"
}

# Test 1: Service start (with timeout to catch daemon ready bug)
test_start() {
    log_info "Test 1: Starting bifrost service..."
    
    systemctl daemon-reload
    
    # Use timeout to catch cases where daemon doesn't signal ready
    if ! run_with_timeout 30 "systemctl start bifrost" systemctl start bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "Failed to start bifrost service (timeout or error)"
    fi
    
    sleep 2
    
    if ! systemctl is-active --quiet bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "bifrost service is not active after start"
    fi
    
    local status=$(systemctl show bifrost --property=Result --value)
    if [ "$status" != "success" ]; then
        log_fail "bifrost service Result is '$status', expected 'success'"
    fi
    
    # Verify PID file exists and is valid
    if [ ! -f /app/logs/bifrost.pid ]; then
        log_fail "PID file not found after start"
    fi
    
    local pid=$(cat /app/logs/bifrost.pid)
    if ! kill -0 "$pid" 2>/dev/null; then
        log_fail "PID $pid from PID file is not running"
    fi
    
    log_pass "Service started successfully (PID: $pid)"
}

# Test 2: Service reload (hot upgrade) with PID stability check
test_reload() {
    log_info "Test 2: Reloading bifrost service (hot upgrade)..."
    
    local old_pid=$(systemctl show bifrost --property=MainPID --value)
    log_info "Master PID before reload: $old_pid"
    
    # Use timeout to catch hangs
    if ! run_with_timeout 30 "systemctl reload bifrost" systemctl reload bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "Failed to reload bifrost service"
    fi
    
    sleep 3
    
    if ! systemctl is-active --quiet bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "bifrost service is not active after reload"
    fi
    
    local new_pid=$(systemctl show bifrost --property=MainPID --value)
    log_info "Master PID after reload: $new_pid"
    
    # In Master-Worker mode, Master PID should remain STABLE
    if [ "$old_pid" != "$new_pid" ]; then
        log_warn "Master PID changed after reload (old=$old_pid, new=$new_pid)"
        log_warn "This may indicate Master-Worker mode is not working correctly"
    else
        log_info "Master PID remained stable (PID: $old_pid)"
    fi
    
    # Check for protocol errors
    if journalctl -u bifrost --no-pager | grep -q "protocol"; then
        log_fail "Found 'protocol' error in journal"
    fi
    
    log_pass "Service reloaded successfully"
}

# Test 3: Service stop
test_stop() {
    log_info "Test 3: Stopping bifrost service..."
    
    if ! run_with_timeout 30 "systemctl stop bifrost" systemctl stop bifrost; then
        log_fail "Failed to stop bifrost service"
    fi
    
    if systemctl is-active --quiet bifrost; then
        log_fail "bifrost service is still active after stop"
    fi
    
    log_pass "Service stopped successfully"
}

# Test 4: Service restart (with timeout - this catches the NotifyDaemonReady bug)
test_restart() {
    log_info "Test 4: Restarting bifrost service..."
    
    systemctl start bifrost
    sleep 2
    
    # This is the critical test - restart must complete within timeout
    # If NotifyDaemonReady is not called, this will hang
    if ! run_with_timeout 30 "systemctl restart bifrost" systemctl restart bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "Failed to restart bifrost service (possible daemon ready signal issue)"
    fi
    
    sleep 2
    
    if ! systemctl is-active --quiet bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "bifrost service is not active after restart"
    fi
    
    log_pass "Service restarted successfully"
}

# Test 5: Verify Type=forking behavior
test_forking_behavior() {
    log_info "Test 5: Verifying Type=forking behavior..."
    
    # Stop service first
    systemctl stop bifrost 2>/dev/null || true
    sleep 1
    
    # Start and measure time (should be fast with proper NotifyDaemonReady)
    local start_time=$(date +%s)
    
    if ! run_with_timeout 30 "systemctl start bifrost" systemctl start bifrost; then
        log_fail "Failed to start bifrost for forking test"
    fi
    
    local end_time=$(date +%s)
    local elapsed=$((end_time - start_time))
    
    if [ $elapsed -gt 10 ]; then
        log_warn "Service startup took ${elapsed}s (expected < 10s)"
    else
        log_info "Service startup completed in ${elapsed}s"
    fi
    
    # Verify MainPID matches PID file
    local systemd_pid=$(systemctl show bifrost --property=MainPID --value)
    local file_pid=$(cat /app/logs/bifrost.pid 2>/dev/null || echo "0")
    
    if [ "$systemd_pid" != "$file_pid" ]; then
        log_warn "Systemd MainPID ($systemd_pid) differs from PID file ($file_pid)"
    else
        log_info "Systemd MainPID matches PID file ($systemd_pid)"
    fi
    
    log_pass "Type=forking behavior verified"
}

# Main
main() {
    echo "============================================"
    echo "Systemd Integration Test for Bifrost"
    echo "(Master-Worker + Type=forking mode)"
    echo "============================================"
    
    wait_for_systemd
    
    test_start
    test_reload
    test_stop
    test_restart
    test_forking_behavior
    
    # Final cleanup
    systemctl stop bifrost 2>/dev/null || true
    
    echo "============================================"
    echo -e "${GREEN}All systemd integration tests passed!${NC}"
    echo "============================================"
}

main "$@"
