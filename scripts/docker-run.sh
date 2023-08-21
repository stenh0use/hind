#!/usr/bin/env bash

container_name="$1"
network_name="$2"
image_name="$3"
shift; shift; shift
args=($@)

container_status=$(docker ps --no-trunc --format json \
    | jq "select(.Names == \"${container_name}\")")

function docker_run {
    # Run docker container
    docker run \
    --detach \
    --init=false \
    --name ${container_name} \
    --network ${network_name} \
    --privileged \
    --restart on-failure \
    --tmpfs /run \
    --tmpfs /tmp \
    --tty \
    --volume /sys/fs/cgroup:/sys/fs/cgroup:ro \
    ${args[@]} \
    ${image_name}:${HIND_VERSION} > /dev/null
    # --cgroupns=private \
}

if [ -z "${container_status}" ]; then
    echo -e "INFO\t container: creating $container_name"
    docker_run $container_name $network_name $image_name ${args[@]}
elif [ "$(jq -rn "$container_status .State")" != "running" ]; then
    echo -e "INFO\t container: starting $container_name"
    docker start $container_name
elif [ "$(jq -rn "$container_status .State")" == "running" ]; then
    echo -e "INFO\t container: $container_name is already running"
fi
