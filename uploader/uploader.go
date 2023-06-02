package uploader

import (
	"os"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"

	"github.com/DggHQ/dggarchiver-config/misc"
	log "github.com/DggHQ/dggarchiver-logger"
	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
)

type SQLiteConfig struct {
	URI string `yaml:"uri"`
	DB  *gorm.DB
}

type LBRYConfig struct {
	URI         string `yaml:"uri"`
	Author      string `yaml:"author"`
	ChannelName string `yaml:"channel_name"`
}

type Uploader struct {
	Verbose bool
	SQLite  SQLiteConfig      `yaml:"sqlite"`
	LBRY    LBRYConfig        `yaml:"lbry"`
	Plugins misc.PluginConfig `yaml:"plugins"`
}

type Config struct {
	Uploader Uploader        `yaml:"uploader"`
	NATS     misc.NATSConfig `yaml:"nats"`
}

func (cfg *Config) Load() {
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

	cfg.Uploader.initialize()

	// NATS Host Name or IP
	if cfg.NATS.Host == "" {
		log.Fatalf("Please set the nats:host config variable and restart the service")
	}
	// NATS Topic Name
	if cfg.NATS.Topic == "" {
		log.Fatalf("Please set the nats:topic config variable and restart the service")
	}
	cfg.NATS.Load()

	log.Debugf("Config loaded successfully")
}

func (uploader *Uploader) initialize() {
	// SQLite
	if uploader.SQLite.URI == "" {
		log.Fatalf("Please set the SQLITE_DB config variable and restart the app")
	}
	uploader.loadSQLite()

	// LBRY
	if uploader.LBRY.URI == "" {
		log.Fatalf("Please set the uploader:lbry:uri config variable and restart the app")
	}
	if uploader.LBRY.Author == "" {
		log.Fatalf("Please set the uploader:lbry:author config variable and restart the app")
	}
	if uploader.LBRY.ChannelName == "" {
		log.Fatalf("Please set the uploader:lbry:channel_name config variable and restart the app")
	}

	// Lua Plugins
	if uploader.Plugins.Enabled {
		if uploader.Plugins.PathToPlugin == "" {
			log.Fatalf("Please set the uploader:plugins:path config variable and restart the service")
		}
	}
}

func (uploader *Uploader) loadSQLite() {
	var err error

	uploader.SQLite.DB, err = gorm.Open(sqlite.Open(uploader.SQLite.URI), &gorm.Config{})
	if err != nil {
		log.Fatalf("Wasn't able to open the SQLite DB: %s", err)
	}

	uploader.SQLite.DB.AutoMigrate(&dggarchivermodel.UploadedVOD{})
}
