#!/usr/bin/env bash

set -e

script_dir="$(dirname "$0")"

missing_images=$(
    "$script_dir"/check-image.sh "hind.consul.server" "hind.vault.server" "hind.nomad.server" "hind.nomad.client"
)

if [ -n "${missing_images}" ]; then
    echo "Error detected missing 'hind' images"
    echo "Please run 'make build' and try again"
    exit 1
fi

network_name="hind"
"$script_dir"/network-create.sh "$network_name"

container_name="hind.consul.server"
image_name="hind.consul.server"
if [ "${LISTEN_LOCALHOST:-0}" -eq 1 ]; then
    args=("-p 127.0.0.1:8500:8500/tcp")
else
    args=("-p 8500:8500/tcp")
fi
"$script_dir"/docker-run.sh "$container_name" "$network_name" "$image_name" "${args[@]}"

container_name="hind.vault.server"
image_name="hind.vault.server"
if [ "${LISTEN_LOCALHOST:-0}" -eq 1 ]; then
    args=("-p 127.0.0.1:8200:8200/tcp")
else
    args=("-p 8200:8200/tcp")
fi
"$script_dir"/docker-run.sh "$container_name" "$network_name" "$image_name" "${args[@]}"

container_name="hind.nomad.server"
image_name="hind.nomad.server"
if [ "${LISTEN_LOCALHOST:-0}" -eq 1 ]; then
    args=("-p 127.0.0.1:4646:4646/tcp")
else
    args=("-p 4646:4646/tcp")
fi
"$script_dir"/docker-run.sh "$container_name" "$network_name" "$image_name" "${args[@]}"

container_name="hind.nomad.client"
image_name="hind.nomad.client"
args=(
    "--device /dev/fuse"
    "-e CILIUM_ENABLED=${CILIUM_ENABLED:-0}"
    "-e CILIUM_IPV4_RANGE=${CILIUM_IPV4_RANGE}"
)

for count in $(seq -f "%02g" "${NOMAD_CLIENT_COUNT:-1}"); do
    "$script_dir"/docker-run.sh "$container_name.$count" "$network_name" "$image_name" "${args[@]}"
done
