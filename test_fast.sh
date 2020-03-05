#!/bin/sh

trap 'handle_sigint' 2

handle_sigint()
{
  echo "Received SIGINT" >&2
  exit 1
}

echo "start"
n=0
while [ "$n" -lt 5 ]; do
  echo $n
  n=$(( n + 1 ))
  sleep 1
done
echo "done"
