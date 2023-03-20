package config

import (
	"database/sql"

	_ "modernc.org/sqlite"

	log "github.com/DggHQ/dggarchiver-logger"
)

type SQLiteConfig struct {
	URI             string `yaml:"uri"`
	DB              *sql.DB
	InsertStatement *sql.Stmt
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

	uploader.SQLite.DB, err = sql.Open("sqlite", uploader.SQLite.URI)
	if err != nil {
		log.Fatalf("Wasn't able to open the SQLite DB: %s", err)
	}

	_, err = uploader.SQLite.DB.Exec("CREATE TABLE IF NOT EXISTS uploaded_vods (id text, pubtime text, title text, starttime text, endtime text, ogthumbnail text, thumbnail text, thumbnailpath text, path text, duration integer, claim text, lbry_name text, lbry_normalized_name text, lbry_permanent_url text);")
	if err != nil {
		log.Fatalf("Wasn't able to create the SQLite table: %s", err)
	}

	uploader.SQLite.InsertStatement, err = uploader.SQLite.DB.Prepare("INSERT INTO uploaded_vods (id, pubtime, title, starttime, endtime, ogthumbnail, thumbnail, thumbnailpath, path, duration, claim, lbry_name, lbry_normalized_name, lbry_permanent_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);")
	if err != nil {
		log.Fatalf("Wasn't able to prepare the insert statement: %s", err)
	}
}
