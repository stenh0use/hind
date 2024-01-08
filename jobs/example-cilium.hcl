job "example_cilium" {
  // This will end up as a label on the Cilium endpoint.
  meta = {
    "cosmonic.io/app_name" = "example"
  }

  group "client" {
    network {
      // This selects the CNI plugin to use.
      // The name after "cni/" should match the conflist that is configured on
      // the Nomad node.
      // See https://developer.hashicorp.com/nomad/docs/job-specification/network#cni.
      mode = "cni/cilium"
    }

    service {
      name         = "example"
      tags         = ["example"]
      address_mode = "alloc"
    }

    task "client" {
      driver = "docker"

      config {
        image = "quay.io/curl/curl:latest"
        args = [
          "sleep",
          "infinity"
        ]
      }

      identity {
        env  = true
        file = true
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}