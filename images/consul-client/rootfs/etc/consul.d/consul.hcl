datacenter = "local"
acl {
  enabled = false
  default_policy = "allow"
}
retry_join = ["hind.consul.server"]
bind_addr = "{{ GetPrivateIP }}"
addresses {
  dns = "0.0.0.0"
  http = "0.0.0.0"
}
recursors = ["127.0.0.11"]
