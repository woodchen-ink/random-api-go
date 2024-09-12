#!/bin/sh

# 如果挂载的 public 目录为空，则从临时位置复制文件
if [ ! "$(ls -A /root/public)" ]; then
    cp -r /tmp/public/* /root/public/
fi

# 启动应用
./random-api
