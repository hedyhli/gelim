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
	path := filepath.Join(xdg.ConfigHome(), "gelim", "config.toml")
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
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
