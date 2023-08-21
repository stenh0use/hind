ARG BASE_IMAGE
FROM $BASE_IMAGE

ENV container=docker

# all non-scripts are 0644 (rw- r-- r--)
COPY --chmod=0644 rootfs/etc/nomad.d/* /etc/nomad.d/
