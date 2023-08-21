#!/usr/bin/env bash

set -e

containers=(
    "hind.consul.server"
    "hind.nomad.server"
    "hind.nomad.client.1"
)

for container in ${containers[@]}; do
    container_name=$container
    container_status=$(docker ps --no-trunc --format json \
    | jq "select(.Names == \"${container_name}\")")

    if [ -n "${container_status}" ]; then
        echo -e "INFO\t conatiner: stopping $container_name"
        docker stop ${container_name} > /dev/null
        docker rm ${container_name} > /dev/null
    else
        echo -e "INFO\t container: $container_name is already stopped"
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
