package config

import (
	"os"
	"time"

	log "github.com/DggHQ/dggarchiver-logger"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"gopkg.in/yaml.v2"
)

func (cfg *Config) loadNats() {
	// Connect to NATS server
	nc, err := nats.Connect(cfg.NATS.Host, nil, nats.PingInterval(20*time.Second), nats.MaxPingsOutstanding(5))
	if err != nil {
		log.Fatalf("Could not connect to NATS server: %s", err)
	}
	log.Infof("Successfully connected to NATS server: %s", cfg.NATS.Host)
	cfg.NATS.NatsConnection = nc
}

func (cfg *Config) Load(service string) {
	var err error

	log.Debugf("Loading the service configuration")
	godotenv.Load()

	configFile := os.Getenv("CONFIG")
	if configFile == "" {
		configFile = "config.yaml"
	}
	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Config load error: %s", err)
	}

	err = yaml.Unmarshal(configBytes, &cfg)
	if err != nil {
		log.Fatalf("YAML unmarshalling error: %s", err)
	}

	switch service {
	case "notifier":
		cfg.Notifier.initialize()
	case "controller":
		cfg.Controller.initialize()
	case "uploader":
		cfg.Uploader.initialize()
	}

	// NATS Host Name or IP
	if cfg.NATS.Host == "" {
		log.Fatalf("Please set the nats:host config variable and restart the service")
	}

	// NATS Topic Name
	if cfg.NATS.Topic == "" {
		log.Fatalf("Please set the nats:topic config variable and restart the service")
	}

	log.Debugf("Config loaded successfully")
}

type Config struct {
	Notifier   Notifier   `yaml:"notifier"`
	Controller Controller `yaml:"controller"`
	Uploader   Uploader   `yaml:"uploader"`
	NATS       NATSConfig `yaml:"nats"`
}
