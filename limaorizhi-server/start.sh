#!/bin/bash
cd /www/wwwroot/limaorizhi-server
[ ! -f ./limaorizhi-server ] && { echo "错误：找不到 limaorizhi-server 二进制文件"; exit 1; }
sed -i '1s/^\xEF\xBB\xBF//; s/\r$//' .env 2>/dev/null
[ -f .env ] && { set -a; source .env; set +a; }
[ -f limaorizhi-server.pid ] && kill $(cat limaorizhi-server.pid) 2>/dev/null; rm -f limaorizhi-server.pid
nohup ./limaorizhi-server > logs.txt 2>&1 &
echo $! > limaorizhi-server.pid
echo "服务已启动（PID: $!）"