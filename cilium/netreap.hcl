job "netreap" {
  datacenters = ["local"]
  priority    = 100
  type        = "system"
  constraint {
    attribute = "${attr.plugins.cni.version.cilium-cni}"
    operator = "is_set"
  }
  group "netreap" {
    restart {
      interval = "10m"
      attempts = 5
      delay = "15s"
      mode = "delay"
    }
    service {
      name = "netreap"
      tags = ["netreap"]
    }
    task "netreap" {
      driver = "docker"
      env {
        NETREAP_CILIUM_CIDR = "10.8.0.0/16"
        NETREAP_DEBUG = "true"
      }
      config {
        image        = "ghcr.io/cosmonic/netreap:0.1.0"
        network_mode = "host"
        # You must be able to mount volumes from the host system so that
        # Netreap can use the Cilium API over a Unix socket.
        # See
        # https://developer.hashicorp.com/nomad/docs/drivers/docker#volumes-1
        # for more information.
        volumes = [
          "/var/run/cilium:/var/run/cilium"
        ]
      }
    }
  }
}