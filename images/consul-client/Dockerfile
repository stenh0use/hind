ARG BASE_IMAGE=consul:testing
FROM $BASE_IMAGE

# all non-scripts are 0644 (rw- r-- r--)
COPY --chmod=0644 rootfs/etc/consul.d/* /etc/consul.d/