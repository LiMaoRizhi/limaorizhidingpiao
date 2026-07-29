#!/bin/bash
cd /www/wwwroot/limaorizhi-server
if [ -f limaorizhi-server.pid ]; then
 PID=$(cat limaorizhi-server.pid); kill $PID 2>/dev/null; sleep 1
 echo "服务已停止（PID: $PID）"; rm -f limaorizhi-server.pid
else
 PIDS=$(pgrep -f limaorizhi-server)
 [ -n "$PIDS" ] && { kill $PIDS; echo "服务已停止"; } || echo "服务未运行"
fi