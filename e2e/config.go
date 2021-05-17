package e2e

const (
	DBdsn           = "postgresql://discovery:discovery@localhost:5432/discovery"
	BrokerURL       = "nats://localhost"
	DockerFile      = "e2e/docker-compose.yml"
	DiscoveryAPIurl = "http://localhost:8080"
	QualityAPIUrl   = "http://localhost:8004"
)
