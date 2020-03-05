#!/bin/sh

echo "start"
n=0
while [ "$n" -lt 60 ]; do
  echo $n
  n=$(( n + 1 ))
  sleep 1
done
echo "done"
