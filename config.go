package main

import (
	"git.sr.ht/~adnano/go-xdg"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Config struct {
	Prompt       string
	MaxRedirects int
	StartURL     string
	LessOpts     string
	SearchURL    string
}

func LoadConfig() (*Config, error) {
	var err error
	var conf Config
	// Defaults
	conf.Prompt = "url/number/cmd; ? for help"
	conf.MaxRedirects = 10
	conf.StartURL = ""
	conf.LessOpts = "-FSXR~ --mouse -P pager (q to quit)"
	conf.SearchURL = "gemini://geminispace.info/search"

	path := filepath.Join(xdg.ConfigHome(), "gelim", "config.toml")
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return &conf, nil
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
