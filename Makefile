DOCKER_CACHE ?= 1

ifeq ($(DOCKER_CACHE), 0)
docker-cache="--no-cache"
endif

.PHONY: up
up:
	docker-compose up -d

.PHONY: down
down:
	docker-compose down

.PHONY: clean
clean:
	docker-compose down -v

.PHONY: build
build: consul nomad

.PHONY: consul
consul: consul-base consul-server consul-client

.PHONY: consul-base
consul-base:
	docker build $(docker-cache) -f ./images/consul-base/Dockerfile -t consul:testing ./images/consul-base

.PHONY: consul-server
consul-server:
	docker build $(docker-cache) -f ./images/consul-server/Dockerfile -t consul-server:testing ./images/consul-server

.PHONY: consul-client
consul-client:
	docker build $(docker-cache) -f ./images/consul-client/Dockerfile -t consul-client:testing ./images/consul-client

.PHONY: nomad
nomad: nomad-base nomad-server nomad-client

.PHONY: nomad-base
nomad-base:
	docker build $(docker-cache) -f ./images/nomad-base/Dockerfile -t nomad:testing ./images/nomad-base

.PHONY: nomad-server
nomad-server:
	docker build $(docker-cache) -f ./images/nomad-server/Dockerfile -t nomad-server:testing ./images/nomad-server

.PHONY: nomad-client
nomad-client:
	docker build $(docker-cache) -f ./images/nomad-client/Dockerfile -t nomad-client:testing ./images/nomad-client
