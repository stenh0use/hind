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
    --cgroupns=private \
    --detach \
    --init=false \
    --name ${container_name} \
    --network ${network_name} \
    --privileged \
    --restart on-failure:1 \
    --tmpfs /run \
    --tmpfs /tmp \
    --tty \
    --security-opt seccomp=unconfined \
    --security-opt apparmor=unconfined \
    --volume /lib/modules:/lib/modules:ro \
    ${args[@]} \
    ${image_name}:${HIND_VERSION} > /dev/null
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
