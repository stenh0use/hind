#!/usr/bin/env bash

set -e

containers=(
    "hind.consul.server"
    "hind.nomad.server"
    "hind.nomad.client"
)

for container in "${containers[@]}"; do
    container_json="$(docker ps -a --no-trunc --format json)"
    container_ids=($(jq \
        -r "select(.Names | startswith(\"${container}\")) | .ID" \
        <<< "$container_json"))
    container_names=($(jq \
        -r "select(.Names | startswith(\"${container}\")) | .Names" \
        <<< "$container_json"))

    if [ ${#container_ids[@]} -gt 0 ]; then
        printf "INFO\t container: stopping %s\n" "${container_names[@]}"
        docker stop "${container_ids[@]}" > /dev/null
        docker rm "${container_ids[@]}" > /dev/null
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
    docker network rm "$network_name" > /dev/null
fi
