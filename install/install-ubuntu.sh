#!/bin/bash

set -e

which taylor > /dev/null
if [ "$?" -eq 1 ]; then
  echo "executable taylor not found"
  exit 1
fi

mkdir -p /var/taylor
chown ubuntu:ubuntu /var/taylor
if [ ! -f /var/taylor/taylor-server-config.json ]; then
  echo "Copy taylor-server-config.json"
  cp taylor-server-config.json /var/taylor/
fi

if [ ! -f /var/taylor/taylor-agent-config.json ]; then
  echo "Copy taylor-agent-config.json"
  cp taylor-agent-config.json /var/taylor/
fi

echo "Copy syslog/30-taylor.conf to /etc/rsyslog.d"
cp syslog/30-taylor.conf /etc/rsyslog.d/
echo "Copy systemd/taylor-server.service /lib/systemd/system"
sudo cp systemd/taylor-server.service /lib/systemd/system/
echo "Copy systemd/taylor-agent.service /lib/systemd/system"
sudo cp systemd/taylor-agent.service /lib/systemd/system/

echo "Please enable imtcp in /etc/rsyslog.conf and run 'systemctl restart rsyslog'"

echo "For taylor server run 'systemctl enable taylor-server.service'"
echo "For taylor agent run  'systemctl enable taylor-agent.service'"
echo "Don't run both :-)"
