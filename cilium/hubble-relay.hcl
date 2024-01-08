job "hubble-relay" {
  priority = 100
  type     = "service"
  constraint {
    attribute = "${attr.plugins.cni.version.cilium-cni}"
    operator  = "is_set"
  }
  group "hubble-relay" {
    service {
      name = "hubble-relay"
      tags = ["hubble-relay"]
    }
    network {
      port "grpc" {
        static = 4245
        to     = 4245
      }
    }
    task "hubble-relay" {
      driver = "docker"
      config {
        network_mode = "host"
        image        = "quay.io/cilium/hubble-relay:v1.13.9"
        volumes = [
          "/var/run/cilium/hubble.sock:/var/run/cilium/hubble.sock"
        ]
        ports = ["grpc"]
        args = [
          "serve",
          "--disable-client-tls",
          "--disable-server-tls",
        ]
      }
    }
  }
}