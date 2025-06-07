#!/bin/sh
set -e

# 等待配置文件出現
CONFIG="/app/config.yaml"
until [ -f "$CONFIG" ]; do
  echo "Waiting for $CONFIG to be mounted..."
  sleep 1
done

# 執行主應用
echo "Starting bifrost application..."
./bifrost