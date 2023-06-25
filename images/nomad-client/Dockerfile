ARG BASE_IMAGE=nomad:testing
FROM $BASE_IMAGE

ENV DOCKER_RELEASES=https://download.docker.com/linux/debian/dists/bullseye/pool/stable
ENV CONTAINERD_VERSION=1.6.9-1
ENV DOCKER_CE_VERSION=24.0.2-1

# all non-scripts are 0644 (rw- r-- r--)
COPY --chmod=0644 rootfs/etc/nomad.d/* /etc/nomad.d/

RUN sed -i "s/^User=.*\$/User=root/g" /etc/systemd/system/nomad.service \
    && sed -i "s/^Group=.*\$/Group=root/g" /etc/systemd/system/nomad.service

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

RUN systemctl enable docker

ENV CNI_RELEASES=https://github.com/containernetworking/plugins/releases
ENV CNI_VERSION=1.3.0

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
    && for file in "bridge" "dhcp" "host-local" "loopback" "firewall" "portmap" "ptp"; do cp $file /opt/cni/bin; done \
    && rm -rf /tmp/build
