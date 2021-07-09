package main

import (
	"git.sr.ht/~adnano/go-xdg"
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
	"io/ioutil"
)

type Config struct {
	Pager        string
	Prompt       string
	MaxRedirects int
}

func LoadConfig() (*Config, error) {
	var err error
	var conf Config
	os.MkdirAll(filepath.Join(xdg.ConfigHome(), "gelim"), 0700)
	path := filepath.Join(xdg.ConfigHome(), "gelim", "config.toml")
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		contents, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
		if _, err = toml.Decode(string(contents), &conf); err != nil {
			return nil, err
		}
	}

	return &conf, nil
}
