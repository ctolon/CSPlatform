#!/bin/bash

set -eo pipefail

NETWORK_NAME='codeserver_net'
SUBNET=172.71.0.0/22
GATEWAY=172.71.0.1

docker network create \
    --driver=bridge \
    --subnet=$SUBNET \
    --gateway=$GATEWAY \
    --attachable \
    --ipv6=false \
    $NETWORK_NAME