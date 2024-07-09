#!/bin/bash

# change to root dirclear
cd $(dirname ${BASH_SOURCE[0]})/..

pid=`ps -ef | grep "0g-monitor" | grep -v grep | awk '{print $2}'`

if [[ "$pid" != "" ]]; then
    echo "Monitoring service already started, pid = $pid"
    exit 1
fi

if ! [ -f .env ]; then
    echo "Error: .env file not found under `pwd`"
    exit 1
fi

source .env
nohup ./0g-monitor --log-disable-color &
