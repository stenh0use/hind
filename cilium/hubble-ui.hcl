job "hubble-ui" {
  datacenters = ["local"]
  priority    = 100
  type        = "service"

  group "hubble-ui" {
    service {
      name = "hubble-ui"
      tags = ["hubble"]
    }
    network {
      port "ui" {
        static = 8080
      }
      port "grpc" {
        static = 8090
      }
    }
    // task "ui" {
    //   driver = "docker"
    //   config {
    //     image = "quay.io/cilium/hubble-ui:v0.12.1"
    //     ports = ["ui"]
    //   }
    // }
    task "backend" {
      driver = "docker"
      config {
        image = "quay.io/cilium/hubble-ui-backend:v0.5.0"
        ports = ["grpc"]
        // volumes = [
        //   "/tmp/config:/root/.kube/config"
        // ]
      }
      env {
        EVENTS_SERVER_PORT = "8090"
        FLOWS_API_ADDR     = "${NOMAD_IP_grpc}:4245"
      }
    }
  }
}