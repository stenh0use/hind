# Hashistack in Docker (hind)
This repository takes inspiration from the Kubernetes in Docker project. First iteration is very rough, consider this a WIP.

Implemented from the stack:
- nomad
- consul

Bits and pieces adapted from:

https://github.com/kubernetes-sigs/kind<br>
https://github.com/multani/docker-nomad<br>
https://github.com/hashicorp/docker-consul<br>

# To run Hind
To build the docker containers
```
make build
```
To run the stack
```
make up
```
To scale the clients in or out
```
make up NOMAD_CLIENT_COUNT=<count>
```
To tear down the stack
```
make down
```
To connect to the web ui for nomad/consul
```
# nomad
http://localhost:4646/ui
# consul
http://localhost:8500/ui
```

# Current limitations
There is no client persistence when running up and down.

# TODO
- add comments for where code is copied
- add sysctls for nomad client and consul??
- add other driver installs? optional java?
