package config

import (
	"os"
	"path/filepath"
)

var DatabaseFile string
var EndScreenFile string
var SecretFile string
var TokenFile string

func LoadConfig() {
	dir, err := os.Executable()
	if err != nil {
		panic(err)
	}

	DatabaseFile = filepath.Join(filepath.Dir(dir), "database.db")
	EndScreenFile = filepath.Join(filepath.Dir(dir), "end.mp4")
	SecretFile = filepath.Join(filepath.Dir(dir), "secret.json")
	TokenFile = filepath.Join(filepath.Dir(dir), "token.json")
}
