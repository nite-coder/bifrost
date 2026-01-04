# Systemd test container based on Ubuntu 22.04 with systemd
FROM ubuntu:22.04

ENV container=docker
ENV DEBIAN_FRONTEND=noninteractive

# Install systemd and cleanup
RUN apt-get update && \
    apt-get install -y systemd systemd-sysv curl && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Remove unnecessary systemd services
RUN rm -f /lib/systemd/system/multi-user.target.wants/* \
    /etc/systemd/system/*.wants/* \
    /lib/systemd/system/local-fs.target.wants/* \
    /lib/systemd/system/sockets.target.wants/*udev* \
    /lib/systemd/system/sockets.target.wants/*initctl* \
    /lib/systemd/system/sysinit.target.wants/systemd-tmpfiles-setup* \
    /lib/systemd/system/systemd-update-utmp*

# Create work directory
WORKDIR /app
RUN mkdir -p /app/logs /app/conf

# Copy bifrost binary and config
COPY bifrost /app/bifrost
COPY test_config.yaml /app/conf/config.yaml
COPY bifrost.service /etc/systemd/system/bifrost.service

# Copy test script
COPY run_test.sh /app/run_test.sh
RUN chmod +x /app/run_test.sh /app/bifrost

# Systemd as PID 1
STOPSIGNAL SIGRTMIN+3
VOLUME ["/sys/fs/cgroup"]

CMD ["/lib/systemd/systemd"]
