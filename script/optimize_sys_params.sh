#!/bin/bash

echo "Optimizing system parameters..."

# Directly modify /etc/sysctl.conf file
echo "Saving settings to /etc/sysctl.conf ..."

echo "vm.swappiness=0" >> /etc/sysctl.conf
echo "vm.dirty_ratio=80" >> /etc/sysctl.conf
echo "vm.dirty_background_ratio=2" >> /etc/sysctl.conf
echo "net.core.somaxconn=262144" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog=262144" >> /etc/sysctl.conf
if [ -f /proc/sys/net/core/netdev_max_backlog ]; then
    echo "net.core.netdev_max_backlog=262144" >> /etc/sysctl.conf
fi
echo "net.ipv4.tcp_max_orphans=262144" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_tw_buckets=2000000" >> /etc/sysctl.conf
echo "net.ipv4.tcp_synack_retries=2" >> /etc/sysctl.conf
echo "net.ipv4.tcp_syncookies=1" >> /etc/sysctl.conf
echo "net.ipv4.conf.all.rp_filter=0" >> /etc/sysctl.conf
echo "net.ipv4.tcp_tw_reuse=1" >> /etc/sysctl.conf
echo "net.ipv4.tcp_timestamps=0" >> /etc/sysctl.conf

echo "Settings saved."

# Reload sysctl configuration
echo "Reloading sysctl configuration..."
sudo sysctl -p

echo "Optimization completed."