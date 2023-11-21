# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# This Dockerfile creates a production release image for the project. This
# downloads the release from releases.hashicorp.com and therefore requires that
# the release is published before building the Docker image.
#
# We don't rebuild the software because we want the exact checksums and
# binary signatures to match the software and our builds aren't fully
# reproducible currently.
ARG BASE_IMAGE=debian:bullseye-slim
FROM $BASE_IMAGE as build

# This is the release of Consul to pull in.
ARG CONSUL_VERSION

# This is the location of the releases.
ENV HASHICORP_RELEASES=https://releases.hashicorp.com

# all scripts are 0755 (rwx r-x r-x)
COPY --chmod=0755 rootfs/usr/local/bin/* /usr/local/bin/

# all non-scripts are 0644 (rw- r-- r--)
COPY --chmod=0644 rootfs/etc/* /etc/
COPY --chmod=0644 rootfs/etc/systemd/system/* /etc/systemd/system/

# Enable systemd.
RUN apt-get update \
    && apt-get install -y systemd systemd-sysv \
    && apt-get clean \
    && find /lib/systemd/system/sysinit.target.wants/ -name "systemd-tmpfiles-setup.service" -delete \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* \
    && rm -rf /lib/systemd/system/multi-user.target.wants/* \
    && rm -rf /etc/systemd/system/*.wants/* \
    && rm -rf /lib/systemd/system/local-fs.target.wants/* \
    && rm -rf /lib/systemd/system/sockets.target.wants/*udev* \
    && rm -rf /lib/systemd/system/sockets.target.wants/*initctl* \
    && echo "ReadKMsg=no" >> /etc/systemd/journald.conf

RUN systemctl enable consul

# Create a consul user and group first so the IDs get set the same way, even as
# the rest of this may change over time.
RUN addgroup consul \
    && adduser --system --group consul

# Set up certificates, base tools, and Consul.
# libc6 is needed to symlink the shared libraries for ARM builds
RUN set -eux \
    && apt update && apt install -y ca-certificates curl gnupg libcap-dev openssl \
       iputils-ping iputils-arping iputils-tracepath jq libc6 iptables tzdata wget unzip \
    && gpg --keyserver keyserver.ubuntu.com --recv-keys C874011F0AB405110D02105534365D9472D7468F \
    && mkdir -p /tmp/build \
    && cd /tmp/build \
    && binArch="$(uname -m)" \
    &&  case "${binArch}" in \
            aarch64) consulArch='arm64' ;; \
            armhf) consulArch='arm' ;; \
            x86) consulArch='386' ;; \
            x86_64) consulArch='amd64' ;; \
            *) echo >&2 "error: unsupported architecture: ${binArch} (see ${HASHICORP_RELEASES}/consul/${CONSUL_VERSION}/)" && exit 1 ;; \
        esac \
    && wget ${HASHICORP_RELEASES}/consul/${CONSUL_VERSION}/consul_${CONSUL_VERSION}_linux_${consulArch}.zip \
    && wget ${HASHICORP_RELEASES}/consul/${CONSUL_VERSION}/consul_${CONSUL_VERSION}_SHA256SUMS \
    && wget ${HASHICORP_RELEASES}/consul/${CONSUL_VERSION}/consul_${CONSUL_VERSION}_SHA256SUMS.sig \
    && gpg --batch --verify consul_${CONSUL_VERSION}_SHA256SUMS.sig consul_${CONSUL_VERSION}_SHA256SUMS \
    && grep consul_${CONSUL_VERSION}_linux_${consulArch}.zip consul_${CONSUL_VERSION}_SHA256SUMS | sha256sum -c \
    && unzip -d /tmp/build consul_${CONSUL_VERSION}_linux_${consulArch}.zip \
    && cp /tmp/build/consul /bin/consul \
    && if [ -f /tmp/build/EULA.txt ]; then mkdir -p /usr/share/doc/consul; mv /tmp/build/EULA.txt /usr/share/doc/consul/EULA.txt; fi \
    && if [ -f /tmp/build/TermsOfEvaluation.txt ]; then mkdir -p /usr/share/doc/consul; mv /tmp/build/TermsOfEvaluation.txt /usr/share/doc/consul/TermsOfEvaluation.txt; fi \
    && cd /tmp \
    && rm -rf /tmp/build \
    && gpgconf --kill all \
    && apt remove -y gnupg openssl \
    && rm -rf /root/.gnupg \
# tiny smoke test to ensure the binary we downloaded runs
    && consul version

# The /consul/data dir is used by Consul to store state. The agent will be started
# with /consul/config as the configuration directory so you can add additional
# config files in that location.
RUN mkdir -p /consul/data \
    && mkdir -p /etc/consul.d \
    && chown -R consul:consul /consul \
    && chown -R consul:consul /etc/consul.d

# set up nsswitch.conf for Go's "netgo" implementation which is used by Consul,
# otherwise DNS supercedes the container's hosts file, which we don't want.
RUN test -e /etc/nsswitch.conf || echo 'hosts: files dns' > /etc/nsswitch.conf

# squash down to one compressed layer, without any lingering whiteout files etc
FROM scratch
COPY --from=build / /
# add metadata, must be done after the squashing
# first tell systemd that it is in docker (it will check for the container env)
# https://systemd.io/CONTAINER_INTERFACE/
ENV container docker
ENV DEBIAN_FRONTEND noninteractive
# Expose the consul data directory as a volume since there's mutable state in there.
VOLUME /consul/data
# Server RPC is used for communication between Consul clients and servers for internal
# request forwarding.
EXPOSE 8300
# Serf LAN and WAN (WAN is used only by Consul servers) are used for gossip between
# Consul agents. LAN is within the datacenter and WAN is between just the Consul
# servers in all datacenters.
EXPOSE 8301 8301/udp 8302 8302/udp
# HTTP and DNS (both TCP and UDP) are the primary interfaces that applications
# use to interact with Consul.
EXPOSE 8500 8600 8600/udp

# systemd exits on SIGRTMIN+3, not SIGTERM (which re-executes it)
# https://bugzilla.redhat.com/show_bug.cgi?id=1201657
STOPSIGNAL SIGRTMIN+3
# NOTE: this is *only* for documentation, the entrypoint is overridden later
ENTRYPOINT [ "/usr/local/bin/consul-entrypoint", "/sbin/init" ]