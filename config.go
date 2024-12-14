package main

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
	UseCertificate      []string
}

// LoadConfig opens the specified configuration file if exists and returns a
// parsed configuration structure
func LoadConfig(path string) (*Config, error) {
	var err error
	var conf Config
	// Defaults
	conf.Prompt = "%U\n>"
	conf.MaxRedirects = 5
	conf.ShowRedirectHistory = true
	conf.StartURL = ""
	// XXX: -R is supposedly better than -r, but -R resets ansi formats on
	// newlines :/
	conf.LessOpts = "-FSXr~ -P pager (q to quit)"
	conf.SearchURL = "gemini://kennedy.gemi.dev/search"
	conf.MaxWidth = 70
	conf.ClipboardCopyCmd = ""

	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return &conf, err
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
	fullURL := u.String()
	if u.Scheme == "gopher" && strings.Contains(fullURL, "%09") {
		parts := strings.Split(fullURL, "%09")
		fullURL = strings.Join(parts[:len(parts)-1], "%09")
	}
	path := u.Path
	if u.Scheme == "gopher" {
		if strings.Contains(path, "\t") {
			parts := strings.Split(path, "\t")
			path = strings.Join(parts[:len(parts)-1], "%09")
		}
		parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
		if len(parts) == 2 {
			path = "/" + parts[1]
		} else {
			path = "/" + parts[0]
		}
	}
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
				prompt += strings.TrimSuffix(strings.TrimPrefix(fullURL, u.Scheme+"://"), "?"+u.RawQuery)
			case 'P':
				if !strings.HasPrefix(path, "/") {
					prompt += "/"
				}
				prompt += path
			case 'p':
				prompt += filepath.Base(path)
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
