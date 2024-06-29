package misc

import (
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/containrrr/shoutrrr/pkg/router"
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

type Notifications struct {
	List       []string `yaml:"list"`
	Conditions []string `yaml:"conditions"`
	Sender     *router.ServiceRouter
}

func (n *Notifications) Enabled() bool {
	return len(n.List) > 0 && len(n.Conditions) > 0
}

func (n *Notifications) Condition(s string) bool {
	return len(n.List) > 0 && len(n.Conditions) > 0 && slices.Contains(n.Conditions, s)
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
