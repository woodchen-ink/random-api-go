#!/bin/sh

# 如果挂载的 public 目录为空，则从临时位置复制文件
if [ ! "$(ls -A /root/data/public)" ]; then
    mkdir -p /root/data/public
    cp -r /tmp/public/* /root/data/public/
fi

# 创建其他必要的目录
mkdir -p /root/data/logs

# 启动应用
./random-api
