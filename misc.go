package config

import "github.com/nats-io/nats.go"

type Flags struct {
	Verbose bool
}

type NATSConfig struct {
	Host           string `yaml:"host"`
	Topic          string `yaml:"topic"`
	NatsConnection *nats.Conn
}

type PluginConfig struct {
	Enabled      bool   `yaml:"enabled"`
	PathToPlugin string `yaml:"path"`
}
