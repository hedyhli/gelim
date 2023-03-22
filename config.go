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

// Config is the configuration structure for gelim
type Config struct {
	Prompt           string
	MaxRedirects     int
	StartURL         string
	LessOpts         string
	SearchURL        string
	Index0Shortcut   int
	LeftMargin       float32
	MaxWidth         int
	ClipboardCopyCmd string
}

// LoadConfig opes the configuration file at $XDG_CONFIG_HOME/gelim/config.toml
// if exists and returns a parsed configuration structure
func LoadConfig() (*Config, error) {
	var err error
	var conf Config
	// Defaults
	conf.Prompt = "%U>"
	conf.MaxRedirects = 10
	conf.StartURL = ""
	// FIXME: -R is supposedly better than -r, but -R resets ansi formats on
	// newlines :/
	conf.LessOpts = "-FSXr~ -P pager (q to quit)"
	conf.SearchURL = "gemini://geminispace.info/search"
	conf.LeftMargin = 0.15
	conf.MaxWidth = 102 // 100 + allowance of 2 ;P
	conf.ClipboardCopyCmd = ""

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
				prompt += strings.TrimSuffix(u.String(), "?"+u.RawQuery)
			case 'u':
				prompt += strings.TrimSuffix(strings.TrimPrefix(u.String(), u.Scheme+"://"), "?"+u.RawQuery)
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
