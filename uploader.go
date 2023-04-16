package config

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

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
	SQLite  SQLiteConfig `yaml:"sqlite"`
	LBRY    LBRYConfig   `yaml:"lbry"`
	Plugins PluginConfig `yaml:"plugins"`
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
