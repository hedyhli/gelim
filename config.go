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
	Prompt              string
	MaxRedirects        int
	ShowRedirectHistory bool
	StartURL            string
	LessOpts            string
	SearchURL           string
	Index0Shortcut      int
	MaxWidth            int
	ClipboardCopyCmd    string
}

// LoadConfig opes the configuration file at $XDG_CONFIG_HOME/gelim/config.toml
// if exists and returns a parsed configuration structure
func LoadConfig() (*Config, error) {
	var err error
	var conf Config
	// Defaults
	conf.Prompt = "%U>"
	conf.MaxRedirects = 5
	conf.ShowRedirectHistory = true
	conf.StartURL = ""
	// XXX: -R is supposedly better than -r, but -R resets ansi formats on
	// newlines :/
	conf.LessOpts = "-FSXr~ -P pager (q to quit)"
	conf.SearchURL = "gemini://kennedy.gemi.dev/search"
	conf.MaxWidth = 90
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

func (c *Client) parsePrompt() string {
	var u *url.URL
	if len(c.history) != 0 {
		u = c.history[len(c.history)-1]
	}
	return BuildPrompt(u, c.conf.Prompt)
}

func BuildPrompt(u *url.URL, promptConf string) (prompt string) {
	percent := false
	for _, char := range promptConf {
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
			case 'H':
				prompt += u.Host
			case 'h':
				prompt += u.Hostname()
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
