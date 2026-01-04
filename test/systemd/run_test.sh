#!/bin/bash
# Systemd Integration Test Script
# This script runs inside the systemd container to test bifrost service

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
log_info() { echo -e "[INFO] $1"; }

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

# Test 1: Service start
test_start() {
    log_info "Test 1: Starting bifrost service..."
    
    systemctl daemon-reload
    
    if ! systemctl start bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "Failed to start bifrost service"
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
    
    log_pass "Service started successfully"
}

# Test 2: Service reload (hot upgrade)
test_reload() {
    log_info "Test 2: Reloading bifrost service (hot upgrade)..."
    
    local old_pid=$(systemctl show bifrost --property=MainPID --value)
    log_info "Old PID: $old_pid"
    
    if ! systemctl reload bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "Failed to reload bifrost service"
    fi
    
    sleep 3
    
    if ! systemctl is-active --quiet bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "bifrost service is not active after reload"
    fi
    
    local new_pid=$(systemctl show bifrost --property=MainPID --value)
    log_info "New PID: $new_pid"
    
    if [ "$old_pid" = "$new_pid" ]; then
        log_fail "PID did not change after reload (old=$old_pid, new=$new_pid)"
    fi
    
    # Check for protocol errors
    if journalctl -u bifrost --no-pager | grep -q "protocol"; then
        log_fail "Found 'protocol' error in journal"
    fi
    
    log_pass "Service reloaded successfully (PID changed from $old_pid to $new_pid)"
}

# Test 3: Service stop
test_stop() {
    log_info "Test 3: Stopping bifrost service..."
    
    if ! systemctl stop bifrost; then
        log_fail "Failed to stop bifrost service"
    fi
    
    if systemctl is-active --quiet bifrost; then
        log_fail "bifrost service is still active after stop"
    fi
    
    log_pass "Service stopped successfully"
}

# Test 4: Service restart
test_restart() {
    log_info "Test 4: Restarting bifrost service..."
    
    systemctl start bifrost
    sleep 2
    
    if ! systemctl restart bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "Failed to restart bifrost service"
    fi
    
    sleep 2
    
    if ! systemctl is-active --quiet bifrost; then
        journalctl -u bifrost --no-pager -n 50
        log_fail "bifrost service is not active after restart"
    fi
    
    log_pass "Service restarted successfully"
}

# Main
main() {
    echo "============================================"
    echo "Systemd Integration Test for Bifrost"
    echo "============================================"
    
    wait_for_systemd
    
    test_start
    test_reload
    test_stop
    test_restart
    
    # Final cleanup
    systemctl stop bifrost 2>/dev/null || true
    
    echo "============================================"
    echo -e "${GREEN}All systemd integration tests passed!${NC}"
    echo "============================================"
}

main "$@"
