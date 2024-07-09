#!/bin/bash

pid=`ps -ef | grep "0g-monitor" | grep -v grep | awk '{print $2}'`

if [[ "$pid" = "" ]]; then
    echo "Monitoring service not started yet"
    exit 1
fi

echo "Terminate monitoring service, pid = $pid"
kill $pid
