#!/usr/bin/env bash

set -e

containers=(
    "hind.consul.server"
    "hind.nomad.server"
    "hind.nomad.client"
)

for container in ${containers[@]}; do
    container_ids=$(docker ps --no-trunc --format json \
        | jq -r "select(.Names | startswith(\"${container}\")) | .ID")

    if [ -n "${container_ids}" ]; then
        echo -e "INFO\t conatiner: stopping $container"
        docker stop ${container_ids} > /dev/null
        docker rm ${container_ids} > /dev/null
    else
        echo -e "INFO\t container: $container is already stopped"
    fi
done

network_name="hind"
network_status=$(docker network ls --format json | jq "select(.Name == \"${network_name}\")")

if [ -z "${network_status}" ]; then
    echo -e "INFO\t network: $network_name has already been removed"
else
    echo -e "INFO\t network: removing $network_name"
    docker network rm $network_name > /dev/null
fi
