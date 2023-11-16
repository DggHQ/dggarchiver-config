package misc

import (
	"log/slog"
	"os"
	"strings"
	"time"

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
		slog.Error("unable to connect to NATS server", slog.Any("err", err))
		os.Exit(1)
	}
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

func SetupSlog(lvl *slog.LevelVar) {
	var (
		h            slog.Handler
		loggerSource bool
	)

	loggerType := strings.ToLower(os.Getenv("LOGGER_TYPE"))
	if loggerSourceString, exists := os.LookupEnv("LOGGER_SOURCE"); exists {
		lc := strings.ToLower(loggerSourceString)
		if lc == "true" || lc == "1" {
			loggerSource = true
		}
	}

	switch loggerType {
	case "json":
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: loggerSource,
			Level:     lvl,
		})
	default:
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: loggerSource,
			Level:     lvl,
		})
	}

	slog.SetDefault(slog.New(h))
}
