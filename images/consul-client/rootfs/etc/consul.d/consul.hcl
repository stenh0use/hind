datacenter = "local"
acl {
  enabled = false
  default_policy = "allow"
}
retry_join = ["consul-server"]
bind_addr = "{{ GetPrivateIP }}"
addresses {
  dns = "0.0.0.0"
  http = "0.0.0.0"
}