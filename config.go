package main

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"git.sr.ht/~adnano/go-xdg"
	"github.com/BurntSushi/toml"
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
	conf.Prompt = "%U>"
	conf.MaxRedirects = 10
	conf.StartURL = ""
	conf.LessOpts = "-FSXR~ -P pager (q to quit)"
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

func (c *Client) parsePrompt() (prompt string) {
	percent := false
	var u *url.URL
	if len(c.history) != 0 {
		u = c.history[len(c.history)-1]
	}
	for _, char := range c.conf.Prompt {
		if char == '%' {
			if percent {
				prompt += "%"
				percent = false
				continue
			}
			percent = true
			continue
		}
		if percent {
			if u == nil {
				percent = false
				continue
			}
			switch char {
			case 'U':
				prompt += strings.TrimRight(u.String(), "?"+u.RawQuery)
			case 'u':
				prompt += strings.TrimRight(strings.TrimLeft(u.String(), u.Scheme+"://"), "?"+u.RawQuery)
			case 'P':
				if !strings.HasPrefix(u.Path, "/") {
					prompt += "/" + u.Path
					break
				}
				prompt += u.Path
			case 'p':
				prompt += filepath.Base(u.Path)
			default:
				prompt += "%" + string(char)
			}
			percent = false
			continue
		}
		prompt += string(char)
	}
	return
}
