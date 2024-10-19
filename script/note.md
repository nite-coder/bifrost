sysctl -w net.ipv4.tcp_low_latency=1
sysctl -w net.ipv4.tcp_fastopen=3
sysctl -w net.ipv4.tcp_slow_start_after_idle=0
sysctl -w net.ipv4.tcp_keepalive_intvl=15
sysctl -w net.ipv4.tcp_keepalive_probes=3
sysctl -w net.ipv4.tcp_keepalive_time=300
sysctl -w net.ipv4.tcp_fin_timeout=5
sysctl -w net.core.busy_read=50
sysctl -w net.core.busy_poll=50
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_rmem="4096 87380 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"


sysctl -w vm.swappiness=0
sysctl -w vm.dirty_ratio=80
sysctl -w vm.dirty_background_ratio=2
sysctl -w net.core.somaxconn=262144
sysctl -w net.ipv4.tcp_max_orphans=262144
sysctl -w net.ipv4.tcp_max_tw_buckets=2000000

# network
sysctl -w net.ipv4.tcp_synack_retries=2
sysctl -w net.ipv4.tcp_syncookies=1
sysctl -w net.ipv4.conf.all.rp_filter=0
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.ipv4.tcp_timestamps=0

