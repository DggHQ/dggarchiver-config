package uploader

import (
	"log/slog"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"

	"github.com/DggHQ/dggarchiver-config/misc"
	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
)

type SQLiteConfig struct {
	URI string `yaml:"uri"`
	DB  *gorm.DB
}

type LBRYConfig struct {
	Enabled     bool
	URI         string `yaml:"uri"`
	Author      string `yaml:"author"`
	ChannelName string `yaml:"channel_name"`
}

type RumbleConfig struct {
	Enabled  bool
	Login    string `yaml:"login"`
	Password string `yaml:"password"`
}

type Uploader struct {
	Verbose   bool
	Platforms struct {
		LBRY   LBRYConfig   `yaml:"lbry"`
		Rumble RumbleConfig `yaml:"rumble"`
	}
	Filters struct {
		List      []string `yaml:"list"`
		Behaviour string   `yaml:"behaviour"`
	} `yaml:"filters"`
	SQLite        SQLiteConfig `yaml:"sqlite"`
	Notifications []string     `yaml:"notifications"`
}

type Config struct {
	*Uploader `yaml:"uploader"`
	NATS      misc.NATSConfig `yaml:"nats"`
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

	if cfg.Uploader.Verbose {
		lvl.Set(slog.LevelDebug)
	}

	cfg.Uploader.initialize()

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

func (uploader *Uploader) initialize() {
	// SQLite
	if uploader.SQLite.URI == "" {
		slog.Error("config variable not set", slog.String("var", "uploader:sqlite:uri"))
		os.Exit(1)
	}
	uploader.loadSQLite()

	if !uploader.Platforms.LBRY.Enabled && !uploader.Platforms.Rumble.Enabled {
		slog.Error("no upload platforms enabled")
		os.Exit(1)
	}

	// LBRY
	if uploader.Platforms.LBRY.Enabled {
		if uploader.Platforms.LBRY.URI == "" {
			slog.Error("config variable not set", slog.String("var", "uploader:platforms:lbry:uri"))
			os.Exit(1)
		}
		if uploader.Platforms.LBRY.Author == "" {
			slog.Error("config variable not set", slog.String("var", "uploader:platforms:lbry:author"))
			os.Exit(1)
		}
		if uploader.Platforms.LBRY.ChannelName == "" {
			slog.Error("config variable not set", slog.String("var", "uploader:platforms:lbry:channel_name"))
			os.Exit(1)
		}
	}

	// Rumble
	if uploader.Platforms.Rumble.Enabled {
		if uploader.Platforms.Rumble.Login == "" {
			slog.Error("config variable not set", slog.String("var", "uploader:platforms:rumble:login"))
			os.Exit(1)
		}
		if uploader.Platforms.Rumble.Password == "" {
			slog.Error("config variable not set", slog.String("var", "uploader:platforms:rumble:password"))
			os.Exit(1)
		}
	}

	if uploader.Filters.Behaviour == "" {
		uploader.Filters.Behaviour = "skip"
	}
}

func (uploader *Uploader) loadSQLite() {
	var err error

	uploader.SQLite.DB, err = gorm.Open(sqlite.Open(uploader.SQLite.URI), &gorm.Config{})
	if err != nil {
		slog.Error("unable to open sqlite db", slog.Any("err", err))
		os.Exit(1)
	}

	if err := uploader.SQLite.DB.AutoMigrate(&dggarchivermodel.UploadedVOD{}); err != nil {
		slog.Error("unable to migrate sqlite db", slog.Any("err", err))
		os.Exit(1)
	}
}
