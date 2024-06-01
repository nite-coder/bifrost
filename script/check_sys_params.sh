#!/bin/bash

echo "Current system parameter values:"

echo "1. Swappiness:"
sysctl vm.swappiness

echo "2. Dirty Ratio:"
sysctl vm.dirty_ratio

echo "3. Dirty Background Ratio:"
sysctl vm.dirty_background_ratio

echo "4. TCP Max SYN Backlog:"
sysctl net.ipv4.tcp_max_syn_backlog

echo "5. Netdev Max Backlog (if exists):"
if [ -f /proc/sys/net/core/netdev_max_backlog ]; then
    sysctl net.core.netdev_max_backlog
else
    echo "netdev_max_backlog does not exist"
fi

echo "6. SOMAXCONN:"
sysctl net.core.somaxconn

echo "7. TCP Max Orphans:"
sysctl net.ipv4.tcp_max_orphans

echo "8. TCP Max TW Buckets:"
sysctl net.ipv4.tcp_max_tw_buckets

echo "9. TCP SYNACK Retries:"
sysctl net.ipv4.tcp_synack_retries

echo "10. TCP Syncookies:"
sysctl net.ipv4.tcp_syncookies

echo "11. RP Filter:"
sysctl net.ipv4.conf.all.rp_filter

echo "12. TCP TW Reuse:"
sysctl net.ipv4.tcp_tw_reuse

echo "13. TCP Timestamps:"
sysctl net.ipv4.tcp_timestamps