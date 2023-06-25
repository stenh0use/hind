datacenter = "local"
server {
  enabled = true
  bootstrap_expect = 1
  default_scheduler_config {
    memory_oversubscription_enabled = true
    scheduler_algorithm = "spread"
  }
}
consul {}