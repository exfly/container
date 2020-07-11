#!/usr/bin/env bash
if [ "$EUID" -ne 0 ]
  then echo "You are not root. Will exit"
  exit
fi

sysctl net.ipv4.conf.all.forwarding=1
iptables -P FORWARD ACCEPT
iptables -t nat -A POSTROUTING -o container0 -j MASQUERADE
# Change the name of the interface, enp0s3 to match that of the
# main interface Ethernet/Wifi you use to connect to the internet
# for routing to work successfully.
iptables -t nat -A POSTROUTING -o enp0s3 -j MASQUERADE
