package misc

import (
	"time"

	log "github.com/DggHQ/dggarchiver-logger"
	"github.com/nats-io/nats.go"
)

type Flags struct {
	Verbose bool
}

type NATSConfig struct {
	Host           string `yaml:"host"`
	Topic          string `yaml:"topic"`
	NatsConnection *nats.Conn
}

func (cfg *NATSConfig) Load() {
	// Connect to NATS server
	nc, err := nats.Connect(cfg.Host, nil, nats.PingInterval(20*time.Second), nats.MaxPingsOutstanding(5))
	if err != nil {
		log.Fatalf("Could not connect to NATS server: %s", err)
	}
	log.Infof("Successfully connected to NATS server: %s", cfg.Host)
	cfg.NatsConnection = nc
}

type PluginConfig struct {
	Enabled      bool   `yaml:"enabled"`
	PathToPlugin string `yaml:"path"`
}

func SumArray(array []int) int {
	result := 0
	for _, v := range array {
		result += v
	}
	return result
}
