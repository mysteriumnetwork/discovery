# Discovery Service

## Main principles and features described
### Features
* Fetch MYST prices from the public API
* Create prices for nodes based on the load and multiplier in the countries
* Listen for proposals from NATS Broker and enhance them with quality metrics and tags
* Run proposal expiration job
* Update price config by Universe Admin
* Use Postgres to save proposals, configs, and country multipliers
* Uses single REDIS instance for storing price configuration

## Architecture & Project structure
### Achitecture
![](/docs/architecture.png "Discovery Service Architecture")


![](/docs/service_blocks.png "Discovery Service Architecture")
### Project structure

#### Envs

##### Discovery

```bash
MYSTERIUM_LOG_MODE=json
PORT=8080
QUALITY_ORACLE_URL=https://testnet3-quality.mysterium.network
QUALITY_CACHE_TTL=20s
BADGER_ADDRESS=https://testnet3-badger.mysterium.network
BROKER_URL=nats://testnet3-broker.mysterium.network
UNIVERSE_JWT_SECRET=Some_Secret
REDIS_ADDRESS=redis:6379
REDIS_DB=0
REDIS_PASS=
```

##### Sidecar

```bash
MYSTERIUM_LOG_MODE=json
QUALITY_ORACLE_URL=https://testnet3-quality.mysterium.network
BROKER_URL=nats://testnet3-broker.mysterium.network
REDIS_ADDRESS=redis:6379
REDIS_DB=0
REDIS_PASS=
GECKO_URL=http://wiremock:8080
COINRANKING_URL=http://wiremock:8080
COINRANKING_TOKEN=Some_Token
```

#### NATS Msg Broker channels

* `*.proposal-register.v3` - Register new Node proposal
* `*.proposal-ping.v3` - Node Ping
* `*.proposal-unregister.v3` - Unregister Node proposal

#### Entries

* `cmd/main.go` - Discovery service
* `sidecar/cmd/main.go` - Sidecar Pricing Parser service

#### Code structure

* `/config` - [Discovery] config parser. Env params
* `/tags` - [Discovery] Tags proposals enhancer for SuperProxy
* `/docs` - [Discovery] Auto generated Swagger for REST API
* `/e2e` - e2e tests
* `/health` - [Discovery] health checker REST API
* `/listener` - [Discovery] NATS listener
* `/price/api.go` - [Discovery] Pricing REST API
* `/price/config.go` - [Discovery, Sidecar] Pricing config
* `/price/market.go` - [Sidecar] Scheduled price fetcher from different APIs (Gecko, Coinmarket)
* `/price/network_load_calculator.go` - [Sidecar] Updating countries multipliers based on load (sessions/providers)
* `/price/price_getter.go` - [Discovery] Fetch prices from Redis
* `/price/price_updater.go` - [Sidecar] Scheduled price updater to Redis
* `/proposal/metrics/echnancer.go` - [Discovery] Proposal quality enhancer
* `/proposal/v3` - [Discovery] Proposal structs
* `/proposal/api.go` - [Discovery] Proposal REST API
* `/proposal/repository.go` - [Discovery] Mappings for Postgres
* `/proposal/service.go` - [Discovery] Proposal service with scheduled expiration job
* `/quality/oracleapi` - [Discovery] Quality Oracle REST API client
* `/quality/service.go` -  [Discovery] Caching layer with BigCache for Quality Oracle responses

## Development

### What toolset needed for development

https://docs.mysterium.network/developers/

### How to build it

`mage build` - Builds Discovery service

`mage buildsidecar` - Builds Sidecar Prices Parser

Or for docker

`docker build -t discovery:local .`

`docker build -t discosidecar:local ./sidecar`

### How to run it

`mage up` - will run docker-compose with all needed services

Or

`docker-compose up` - this should fire up all discovery service dependencies and the discovery it'self. (use --force-recreate to rebuild discovery image)

And if you wish to run discovery service from your IDE, then idea is to use

`docker-compose up dev` - This should only fire up dependencies (redis, etc...) for the discovery (or discosidecar).

### How to test it locally

`mage e2edev`

### Generate custom marshaller

`easyjson -all -output_filename proposal/v3/proposal_json.go proposal/v3/proposal.go`
`easyjson -all -output_filename proposal/v3/metadata_json.go proposal/v3/metadata.go`

## API

Docs: http://localhost:8080/swagger/index.html

API: http://localhost:8080/api/v3
