#!/usr/bin/env bash

set -e

network_name=$1
network_status=$(docker network ls --format json | jq "select(.Name == \"${network_name}\")")

function create_network {
    echo -e "INFO\t network: creating $network_name"
    docker network create ${network_name} > /dev/null
}

if [ -z "${network_status}" ]; then
    create_network;
else
    echo -e "INFO\t network: $network_name already exists"
fi
