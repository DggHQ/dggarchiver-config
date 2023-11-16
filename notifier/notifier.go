package notifier

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/DggHQ/dggarchiver-config/misc"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"gopkg.in/yaml.v2"
)

var (
	ErrPriorityNotSet    = errors.New("priority not set for every enabled platform")
	ErrPriorityNotUnique = errors.New("some priority is not a unique number from 1 to <num of enabled platforms>")
)

type Kick struct {
	Enabled        bool
	Downloader     string `yaml:"downloader"`
	Priority       int    `yaml:"restream_priority"`
	Channel        string `yaml:"channel"`
	HealthCheck    string `yaml:"healthcheck"`
	ScraperRefresh int    `yaml:"scraper_refresh"`
	ProxyURL       string `yaml:"proxy_url"`
}

type Rumble struct {
	Enabled        bool
	Downloader     string `yaml:"downloader"`
	Priority       int    `yaml:"restream_priority"`
	Channel        string `yaml:"channel"`
	HealthCheck    string `yaml:"healthcheck"`
	ScraperRefresh int    `yaml:"scraper_refresh"`
}

type YouTube struct {
	Enabled        bool
	Downloader     string `yaml:"downloader"`
	Priority       int    `yaml:"restream_priority"`
	Channel        string `yaml:"channel"`
	HealthCheck    string `yaml:"healthcheck"`
	ScraperRefresh int    `yaml:"scraper_refresh"`
	APIRefresh     int    `yaml:"api_refresh"`
	GoogleCred     string `yaml:"google_credentials"`
	Service        *youtube.Service
}

type Notifier struct {
	Verbose   bool
	Platforms struct {
		YouTube YouTube `yaml:"youtube"`
		Rumble  Rumble  `yaml:"rumble"`
		Kick    Kick    `yaml:"kick"`
	}
	Plugins misc.PluginConfig `yaml:"plugins"`
}

type Config struct {
	Notifier Notifier        `yaml:"notifier"`
	NATS     misc.NATSConfig `yaml:"nats"`
}

func New() *Config {
	var (
		err error
		cfg = Config{}
		lvl slog.LevelVar
	)

	misc.SetupSlog(&lvl)

	_ = godotenv.Load()

	configFile := os.Getenv("CONFIG")
	if configFile == "" {
		configFile = "config.yaml"
	}
	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		slog.Error("unable to load config", slog.Any("err", err))
		os.Exit(1)
	}

	err = yaml.Unmarshal(configBytes, &cfg)
	if err != nil {
		slog.Error("unable to unmarshall config yaml", slog.Any("err", err))
		os.Exit(1)
	}

	if cfg.Notifier.Verbose {
		lvl.Set(slog.LevelDebug)
	}

	cfg.Notifier.initialize()

	// NATS Host Name or IP
	if cfg.NATS.Host == "" {
		slog.Error("config variable not set", slog.String("var", "nats:host"))
		os.Exit(1)
	}
	// NATS Topic Name
	if cfg.NATS.Topic == "" {
		slog.Error("config variable not set", slog.String("var", "nats:topic"))
		os.Exit(1)
	}
	cfg.NATS.Load()

	return &cfg
}

func (notifier *Notifier) validatePlatforms() bool {
	var enabledPlatforms int
	platformsValue := reflect.ValueOf(notifier.Platforms)
	platformsFields := reflect.VisibleFields(reflect.TypeOf(notifier.Platforms))
	for _, field := range platformsFields {
		if platformsValue.FieldByName(field.Name).FieldByName("Enabled").Bool() {
			enabledPlatforms++
		}
	}
	return enabledPlatforms > 0
}

func (notifier *Notifier) validatePriority() error {
	var platformPriority []int
	var numOfEnabledPlatforms int
	platformsValue := reflect.ValueOf(notifier.Platforms)
	platformsFields := reflect.VisibleFields(reflect.TypeOf(notifier.Platforms))
	for _, field := range platformsFields {
		if platformsValue.FieldByName(field.Name).FieldByName("Enabled").Bool() {
			numOfEnabledPlatforms++
			if platformsValue.FieldByName(field.Name).FieldByName("Priority").Int() > 0 {
				platformPriority = append(platformPriority, int(platformsValue.FieldByName(field.Name).FieldByName("Priority").Int()))
			}
		}
	}
	if misc.SumArray(platformPriority) == 0 {
		return nil
	}
	sort.Ints(platformPriority)
	if len(platformPriority) != numOfEnabledPlatforms {
		return ErrPriorityNotSet
	}
	for i := 0; i < numOfEnabledPlatforms; i++ {
		if platformPriority[i] != i+1 {
			return ErrPriorityNotUnique
		}
	}
	return nil
}

func (notifier *Notifier) initialize() {
	if !notifier.validatePlatforms() {
		slog.Error("no platforms enabled")
		os.Exit(1)
	}

	if err := notifier.validatePriority(); err != nil {
		slog.Error("unable to validate platform priority", slog.Any("err", err))
		os.Exit(1)
	}

	// YouTube
	if notifier.Platforms.YouTube.Enabled {
		if notifier.Platforms.YouTube.GoogleCred == "" {
			slog.Error("config variable not set", slog.String("var", "notifier:platform:youtube:google_credentials"))
			os.Exit(1)
		}
		if notifier.Platforms.YouTube.Channel == "" {
			slog.Error("config variable not set", slog.String("var", "notifier:platform:youtube:channel"))
			os.Exit(1)
		}
		if notifier.Platforms.YouTube.ScraperRefresh == 0 && notifier.Platforms.YouTube.APIRefresh == 0 {
			slog.Error("config variable not set",
				slog.String("var", "notifier:platform:youtube:scraper_refresh"),
				slog.String("var", "notifier:platform:youtube:api_refresh"),
			)
			os.Exit(1)
		}
		if notifier.Platforms.YouTube.Downloader == "" {
			notifier.Platforms.YouTube.Downloader = "yt-dlp"
		}
		notifier.createGoogleClients()
	}

	// Rumble
	if notifier.Platforms.Rumble.Enabled {
		if notifier.Platforms.Rumble.Channel == "" {
			slog.Error("config variable not set", slog.String("var", "notifier:platform:rumble:channel"))
			os.Exit(1)
		}
		if notifier.Platforms.Rumble.ScraperRefresh == 0 {
			slog.Error("config variable not set", slog.String("var", "notifier:platform:rumble:scraper_refresh"))
			os.Exit(1)
		}
		if notifier.Platforms.Rumble.Downloader == "" {
			notifier.Platforms.Rumble.Downloader = "yt-dlp"
		}
	}

	// Kick
	if notifier.Platforms.Kick.Enabled {
		if notifier.Platforms.Kick.Channel == "" {
			slog.Error("config variable not set", slog.String("var", "notifier:platform:kick:channel"))
			os.Exit(1)
		}
		if notifier.Platforms.Kick.ScraperRefresh == 0 {
			slog.Error("config variable not set", slog.String("var", "notifier:platform:kick:scraper_refresh"))
			os.Exit(1)
		}
		if notifier.Platforms.Kick.Downloader == "" {
			notifier.Platforms.Kick.Downloader = "yt-dlp"
		}
	}

	// Lua Plugins
	if notifier.Plugins.Enabled {
		if notifier.Plugins.PathToPlugin == "" {
			slog.Error("config variable not set", slog.String("var", "notifier:plugins:path"))
			os.Exit(1)
		}
	}
}

func (notifier *Notifier) createGoogleClients() {
	ctx := context.Background()

	credpath := filepath.Join(".", notifier.Platforms.YouTube.GoogleCred)
	b, err := os.ReadFile(credpath)
	if err != nil {
		slog.Error("unable to read client secret file", slog.Any("err", err))
		os.Exit(1)
	}

	googleCfg, err := google.JWTConfigFromJSON(b, "https://www.googleapis.com/auth/youtube.readonly")
	if err != nil {
		slog.Error("unable to parse client secret file", slog.Any("err", err))
		os.Exit(1)
	}
	client := googleCfg.Client(ctx)

	notifier.Platforms.YouTube.Service, err = youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		slog.Error("unable to retrieve youtube client", slog.Any("err", err))
		os.Exit(1)
	}
}
