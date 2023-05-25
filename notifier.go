package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	log "github.com/DggHQ/dggarchiver-logger"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type Kick struct {
	Enabled        bool
	Downloader     string `yaml:"downloader"`
	Priority       int    `yaml:"restream_priority"`
	Channel        string `yaml:"channel"`
	HealthCheck    string `yaml:"healthcheck"`
	ScraperRefresh int    `yaml:"scraper_refresh"`
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
	Plugins PluginConfig `yaml:"plugins"`
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
	if sumArray(platformPriority) == 0 {
		return nil
	}
	sort.Ints(platformPriority)
	if len(platformPriority) != numOfEnabledPlatforms {
		return errors.New("Please check if the priority has been set for every enabled platform")
	}
	for i := 0; i < numOfEnabledPlatforms; i++ {
		if platformPriority[i] != i+1 {
			return errors.New("Please check if priority for every enabled platform is a unique number from 1 to <num of enabled platforms>")
		}
	}
	return nil
}

func (notifier *Notifier) initialize() {
	if !notifier.validatePlatforms() {
		log.Fatalf("Please enable at least one platform and restart the service")
	}

	if err := notifier.validatePriority(); err != nil {
		log.Fatalf(err.Error())
	}

	// YouTube
	if notifier.Platforms.YouTube.Enabled {
		if notifier.Platforms.YouTube.GoogleCred == "" {
			log.Fatalf("Please set the notifier:platform:youtube:google_credentials config variable and restart the service")
		}
		if notifier.Platforms.YouTube.Channel == "" {
			log.Fatalf("Please set the notifier:platform:youtube:channel config variable and restart the service")
		}
		if notifier.Platforms.YouTube.ScraperRefresh == 0 && notifier.Platforms.YouTube.APIRefresh == 0 {
			log.Fatalf("Please set the notifier:platform:youtube:scraper_refresh or the notifier:platform:youtube:api_refresh config variable and restart the service")
		}
		if notifier.Platforms.YouTube.Downloader == "" {
			notifier.Platforms.YouTube.Downloader = "yt-dlp"
		}
		notifier.createGoogleClients()
	}

	// Rumble
	if notifier.Platforms.Rumble.Enabled {
		if notifier.Platforms.Rumble.Channel == "" {
			log.Fatalf("Please set the notifier:platform:rumble:channel config variable and restart the service")
		}
		if notifier.Platforms.Rumble.ScraperRefresh == 0 {
			log.Fatalf("Please set the notifier:platform:rumble:scraper_refresh config variable and restart the service")
		}
		if notifier.Platforms.Rumble.Downloader == "" {
			notifier.Platforms.Rumble.Downloader = "yt-dlp"
		}
	}

	// Kick
	if notifier.Platforms.Kick.Enabled {
		if notifier.Platforms.Kick.Channel == "" {
			log.Fatalf("Please set the notifier:platform:kick:channel config variable and restart the service")
		}
		if notifier.Platforms.Kick.ScraperRefresh == 0 {
			log.Fatalf("Please set the notifier:platform:kick:scraper_refresh config variable and restart the service")
		}
		if notifier.Platforms.Kick.Downloader == "" {
			notifier.Platforms.Kick.Downloader = "yt-dlp"
		}
	}

	// Lua Plugins
	if notifier.Plugins.Enabled {
		if notifier.Plugins.PathToPlugin == "" {
			log.Fatalf("Please set the notifier:plugins:path config variable and restart the service")
		}
	}
}

func (notifier *Notifier) createGoogleClients() {
	log.Debugf("Creating Google API clients")

	ctx := context.Background()

	credpath := filepath.Join(".", notifier.Platforms.YouTube.GoogleCred)
	b, err := os.ReadFile(credpath)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	googleCfg, err := google.JWTConfigFromJSON(b, "https://www.googleapis.com/auth/youtube.readonly")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := googleCfg.Client(ctx)

	notifier.Platforms.YouTube.Service, err = youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve YouTube client: %v", err)
	}

	log.Debugf("Created Google API clients successfully")
}
