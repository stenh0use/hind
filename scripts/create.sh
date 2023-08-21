#!/usr/bin/env bash

set -e

script_dir="$(dirname "$0")"

network_name="hind"
$script_dir/network-create.sh $network_name

container_name="hind.consul.server"
image_name="hind.consul.server"
args=("-p 8500:8500")
$script_dir/docker-run.sh $container_name $network_name $image_name ${args[@]}

container_name="hind.nomad.server"
image_name="hind.nomad.server"
args=("-p 4646:4646")
$script_dir/docker-run.sh $container_name $network_name $image_name ${args[@]}

container_name="hind.nomad.client.1"
image_name="hind.nomad.client"
args=(
    "--security-opt seccomp=unconfined"
    "--security-opt apparmor=unconfined"
    "--volume /lib/modules:/lib/modules:ro"
    "-e CILIUM_ENABLED=${CILIUM_ENABLED:-0}"
    "-e CILIUM_IPV4_RANGE=${CILIUM_IPV4_RANGE}"
)
$script_dir/docker-run.sh $container_name $network_name $image_name ${args[@]}
