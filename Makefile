DOCKER_CACHE ?= 1

CILIUM_IPV4_RANGE ?= 10.8.0.0/16
CILIUM_ENABLED ?= 0
NOMAD_CLIENT_COUNT ?= 1

BASE_IMAGE = debian:bullseye-slim
CONSUL_VERSION ?= 1.18.1
NOMAD_VERSION ?= 1.7.6
VAULT_VERSION ?= 1.15.4
HIND_VERSION = 0.2.0

ifeq ($(DOCKER_CACHE), 0)
docker-cache="--no-cache"
endif

.PHONY: up
up:
	@HIND_VERSION=$(HIND_VERSION) \
	CILIUM_ENABLED=$(CILIUM_ENABLED) \
	CILIUM_IPV4_RANGE=$(CILIUM_IPV4_RANGE) \
	NOMAD_CLIENT_COUNT=$(NOMAD_CLIENT_COUNT) \
	./scripts/create.sh

.PHONY: down
down:
	@./scripts/destroy.sh

.PHONY: build
build: consul vault nomad

.PHONY: consul
consul: consul-base consul-server consul-client

.PHONY: consul-base
consul-base:
	@docker build $(docker-cache) \
	--build-arg="BASE_IMAGE=$(BASE_IMAGE)" \
	--build-arg="CONSUL_VERSION=$(CONSUL_VERSION)" \
	-f ./images/consul-base/Dockerfile \
	-t hind.consul:$(HIND_VERSION) \
	./images/consul-base

.PHONY: consul-server
consul-server:
	@docker build $(docker-cache) \
	--build-arg="BASE_IMAGE=hind.consul:$(HIND_VERSION)" \
	-f ./images/consul-server/Dockerfile \
	-t hind.consul.server:$(HIND_VERSION) \
	./images/consul-server

.PHONY: consul-client
consul-client:
	@docker build $(docker-cache) \
	--build-arg="BASE_IMAGE=hind.consul:$(HIND_VERSION)" \
	-f ./images/consul-client/Dockerfile \
	-t hind.consul.client:$(HIND_VERSION) \
	./images/consul-client

.PHONY: nomad
nomad: nomad-base nomad-server nomad-client

.PHONY: nomad-base
nomad-base:
	@docker build $(docker-cache) \
	--build-arg="BASE_IMAGE=hind.consul.client:$(HIND_VERSION)" \
	--build-arg="NOMAD_VERSION=$(NOMAD_VERSION)" \
	-f ./images/nomad-base/Dockerfile \
	-t hind.nomad:$(HIND_VERSION) \
	./images/nomad-base

.PHONY: nomad-server
nomad-server:
	@docker build $(docker-cache) \
	--build-arg="BASE_IMAGE=hind.nomad:$(HIND_VERSION)" \
	-f ./images/nomad-server/Dockerfile \
	-t hind.nomad.server:$(HIND_VERSION) \
	./images/nomad-server

.PHONY: nomad-client
nomad-client:
	@docker build $(docker-cache) \
	--build-arg="BASE_IMAGE=hind.nomad:$(HIND_VERSION)" \
	-f ./images/nomad-client/Dockerfile \
	-t hind.nomad.client:$(HIND_VERSION) \
	./images/nomad-client

.PHONY: vault
vault: vault-server

.PHONY: vault-server
vault-server:
	@docker build $(docker-cache) \
	--build-arg="BASE_IMAGE=hind.consul.client:$(HIND_VERSION)" \
	--build-arg="VAULT_VERSION=$(VAULT_VERSION)" \
	-f ./images/vault-server/Dockerfile \
	-t hind.vault.server:$(HIND_VERSION) \
	./images/vault-server
