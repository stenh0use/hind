datacenter = "local"
client {
  enabled = true
}
consul {}
plugin "docker" {
  config {
    volumes {
      enabled = true
    }
  }
}