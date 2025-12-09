# Advanced Networking with Cilium

> **⚠️ Note:** Cilium support is currently non-functional and under development. This documentation is preserved for reference but the feature is not yet working in the current version.

The following setup is based on the Cosmonic [blog post](https://cosmonic.com/blog/engineering/netreap-a-practical-guide-to-running-cilium-in-nomad) about running Cilium in Nomad.

## Enable Cilium

Start a cluster with Cilium CNI enabled:

```bash
./bin/hind start --cni=cilium
```

Check Cilium health status (may take 2-5 minutes to become fully healthy):

```bash
# Access the first client node
docker exec hind.default.nomad.client.01 cilium status
```

Wait for the output to show `Cluster health: 1/1 reachable` with passing health checks.

Restart the Nomad service to fully integrate with Cilium:

```bash
docker exec hind.default.nomad.client.01 systemctl restart nomad
```

## Deploy Netreap

Netreap watches Consul for network policies and applies them via Cilium:

```bash
nomad run cilium/netreap.hcl
```

Apply an allow-all policy:

```bash
consul kv put netreap.io/policy @cilium/policy-allow-all.json
```

## Test Network Policies

Deploy a test workload:

```bash
nomad run jobs/example-cilium.hcl
```

Test connectivity (should succeed with allow-all policy):

```bash
nomad exec -i \
    -t $(curl localhost:4646/v1/job/example_cilium/allocations 2>/dev/null \
    | jq -r '.[0].ID') \
    curl google.com -v
```

Apply a deny policy:

```bash
consul kv put netreap.io/policy @cilium/policy-blocked-egress.json
```

Test connectivity again (should now be blocked):

```bash
nomad exec -i \
    -t $(curl localhost:4646/v1/job/example_cilium/allocations 2>/dev/null \
    | jq -r '.[0].ID') \
    curl google.com -v
```

## Hubble Relay

The [Hubble Relay](https://docs.cilium.io/en/stable/internals/hubble/#hubble-relay) provides observability into network flows:

```bash
nomad run cilium/hubble-relay.hcl
```

Check Hubble health:

```bash
docker exec hind.default.nomad.client.01 hubble status
# Expected output:
# Healthcheck (via localhost:4245): Ok
# Current/Max Flows: 485/4,095 (11.84%)
# Flows/s: 2.45
# Connected Nodes: 1/1
```
