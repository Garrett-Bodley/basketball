package config

import (
	"os"
	"path/filepath"
)

var DatabaseFile string

func LoadConfig() {
	dir, err := os.Executable()
	if err != nil {
		panic(err)
	}

	DatabaseFile = filepath.Join(filepath.Dir(dir), "db/database.db")
}
