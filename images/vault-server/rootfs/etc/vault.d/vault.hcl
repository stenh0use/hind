ui            = true
cluster_addr  = "http://127.0.0.1:8201"
api_addr      = "http://127.0.0.1:8200"
disable_mlock = true

storage "raft" {
  path = "/vault/data"
}

service_registration "consul" {
  address = "127.0.0.1:8500"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = true
}
