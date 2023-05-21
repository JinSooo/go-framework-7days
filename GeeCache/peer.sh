#!/bin/bash
# trap 命令用于在 shell 脚本退出时，删掉临时文件，结束子进程
trap "rm main;kill 0" EXIT

go build -o main
./main -port=8001 &
./main -port=8002 &
# 8003 cache 和9999 api两个服务是在一个里面的，让8003的缓存充当api服务的本地缓存服务器
./main -port=8003 -api=1 &

sleep 2
echo ">>> start test"
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &

# sleep 3600
#  使用read命令达到类似bat中的pause命令效果
echo 按任意键继续
read -n 1
echo 继续运行