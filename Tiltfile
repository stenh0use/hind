# Use Docker compose to manage resources
docker_compose("./docker-compose.yml")

# Specify resources for each container. Groups them and provides friendly names.
dc_resource(
    name = "consul-server",
    new_name = "Consul Server",
    labels = ["Consul"]
)

dc_resource(
    name = "nomad-server",
    new_name = "Nomad Server",
    labels = ["Nomad"],
)

dc_resource(
    name = "nomad-client",
    new_name = "Nomad Client(s)",
    labels = ["Nomad"]
)

# Define build configs for each Docker Image
# This will build the image before running compose, and
# automatically rebuild the image when the Docker context files change.
custom_build(
    "consul-server", 
    "make consul-server",
    ["images/consul-base", "images/consul-server"],
     tag="testing"
)

custom_build(
    "nomad-client", 
    "make nomad-client",
    ["images/nomad-base", "images/nomad-client"],
    tag="testing"
)

custom_build(
    "nomad-server", 
    "make nomad-server",
    ["images/nomad-base", "images/nomad-server"],
    tag="testing"
)