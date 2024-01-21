#!/usr/bin/env bash

set -e

script_dir="$(dirname "$0")"

missing_images=$(
    $script_dir/check-image.sh "hind.consul.server" "hind.nomad.server" "hind.nomad.client"
)

if [ -n "${missing_images}" ]; then
    echo "Error detected missing 'hind' images"
    echo "Please run 'make build' and try again"
    exit 1
fi

network_name="hind"
$script_dir/network-create.sh $network_name

container_name="hind.consul.server"
image_name="hind.consul.server"
args=("-p 127.0.0.1:8500:8500/tcp")
$script_dir/docker-run.sh $container_name $network_name $image_name ${args[@]}

container_name="hind.nomad.server"
image_name="hind.nomad.server"
args=("-p 127.0.0.1:4646:4646/tcp")
$script_dir/docker-run.sh $container_name $network_name $image_name ${args[@]}

container_name="hind.nomad.client"
image_name="hind.nomad.client"
args=(
    "--device /dev/fuse"
    "-e CILIUM_ENABLED=${CILIUM_ENABLED:-0}"
    "-e CILIUM_IPV4_RANGE=${CILIUM_IPV4_RANGE}"
)

for count in $(seq -f "%02g" ${NOMAD_CLIENT_COUNT:-1}); do
    $script_dir/docker-run.sh $container_name.$count $network_name $image_name ${args[@]}
done
