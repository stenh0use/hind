ARG BASE_IMAGE
FROM $BASE_IMAGE

ENV DOCKER_RELEASES=https://download.docker.com/linux/debian/dists/bullseye/pool/stable
ARG CONTAINERD_VERSION=1.6.26-1
ARG DOCKER_CE_VERSION=24.0.7-1
ENV container=docker

# all non-scripts are 0644 (rw- r-- r--)
COPY --chmod=0644 rootfs/etc/nomad.d/* /etc/nomad.d/
# all scripts are 0755 (rwx r-x r-x)
COPY --chmod=0755 rootfs/usr/local/bin/* /usr/local/bin/

# copy unit files and service configs
COPY --chmod=0644 rootfs/etc/systemd/system/* /etc/systemd/system/
COPY --chmod=0644 rootfs/etc/cilium/* /etc/cilium/
COPY --chmod=0644 rootfs/etc/docker/* /etc/docker/

RUN sed -i "s/^User=.*\$/User=root/g" /etc/systemd/system/nomad.service \
    && sed -i "s/^Group=.*\$/Group=root/g" /etc/systemd/system/nomad.service

# install docker-ce and containerd
RUN set -eux \
    && apt install -y iproute2 fuse-overlayfs \
    && mkdir -p /tmp/build \
    && cd /tmp/build \
    && binArch="$(uname -m)" \
    &&  case "${binArch}" in \
            aarch64) nomadArch='arm64' ;; \
            armhf) nomadArch='armhf' ;; \
            x86_64) nomadArch='amd64' ;; \
            *) echo >&2 "error: unsupported architecture: ${binArch} (see ${DOCKER_RELEASES})" && exit 1 ;; \
        esac \
    && wget ${DOCKER_RELEASES}/${nomadArch}/containerd.io_${CONTAINERD_VERSION}_${nomadArch}.deb \
    && wget ${DOCKER_RELEASES}/${nomadArch}/docker-ce_${DOCKER_CE_VERSION}~debian.11~bullseye_${nomadArch}.deb \
    && wget ${DOCKER_RELEASES}/${nomadArch}/docker-ce-cli_${DOCKER_CE_VERSION}~debian.11~bullseye_${nomadArch}.deb \
    && dpkg -i ./containerd.io_${CONTAINERD_VERSION}_${nomadArch}.deb \
       ./docker-ce_${DOCKER_CE_VERSION}~debian.11~bullseye_${nomadArch}.deb \
       ./docker-ce-cli_${DOCKER_CE_VERSION}~debian.11~bullseye_${nomadArch}.deb \
    && cd /tmp \
    && rm -rf /tmp/build

RUN systemctl enable docker.service \
    && systemctl enable containerd.service

# install standard CNI plugins
ENV CNI_RELEASES=https://github.com/containernetworking/plugins/releases
ARG CNI_VERSION=1.3.0

RUN set -eux \
    && mkdir -p /tmp/build \
    && cd /tmp/build \
    && binArch="$(uname -m)" \
    &&  case "${binArch}" in \
            aarch64) nomadArch='arm64' ;; \
            x86_64) nomadArch='amd64' ;; \
            *) echo >&2 "error: unsupported architecture: ${binArch} (see ${DOCKER_RELEASES})" && exit 1 ;; \
        esac \
    && wget ${CNI_RELEASES}/download/v${CNI_VERSION}/cni-plugins-linux-${nomadArch}-v1.3.0.tgz \
    && tar -xvf cni-plugins-linux-${nomadArch}-v1.3.0.tgz \
    && mkdir -p /opt/cni/bin \
    && mkdir -p /opt/cni/config \
    && for file in "bridge" "dhcp" "host-local" "loopback" "firewall" "portmap" "ptp"; do cp $file /opt/cni/bin; done \
    && rm -rf /tmp/build

# install cilium cni plugin and service
ARG CILIUM_SERVICE=disable

RUN mkdir -p /opt/cilium/bin \
    && mkdir -p /opt/cilium/config
COPY --from=cilium/cilium:v1.13.9 /opt/cni/bin/cilium-cni /opt/cilium/bin/cilium-cni
COPY --from=cilium/cilium:v1.13.9 /usr/bin/cilium* /usr/local/bin/
COPY --from=cilium/cilium:v1.13.9 /usr/bin/hubble /usr/local/bin/
COPY --chmod=0644 rootfs/opt/cni/config/* /opt/cilium/config/

RUN systemctl ${CILIUM_SERVICE} cilium-mounts \
    && systemctl ${CILIUM_SERVICE} cilium
