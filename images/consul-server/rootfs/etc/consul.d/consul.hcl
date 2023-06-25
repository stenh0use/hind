datacenter = "local"
server = true
acl {
  enabled = false
  default_policy = "allow"
}
ui_config {
  enabled = true
}
client_addr = "0.0.0.0"
bind_addr = "{{ GetPrivateIP }}"
bootstrap_expect = 1
