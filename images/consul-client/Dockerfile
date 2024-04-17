ARG BASE_IMAGE
FROM $BASE_IMAGE

ENV container docker

RUN mkdir -p /etc/systemd/resolved.conf.d

# all non-scripts are 0644 (rw- r-- r--)
COPY --chmod=0644 rootfs/etc/consul.d/* /etc/consul.d/
COPY --chmod=0644 rootfs/etc/systemd/resolved.conf.d/* /etc/systemd/resolved.conf.d/

RUN systemctl enable systemd-resolved.service
