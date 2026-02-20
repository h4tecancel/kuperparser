package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProxyConfig struct {
	Mode               string   `yaml:"mode"` // disabled|list|rotation
	List               []string `yaml:"list"`
	RotationURL        string   `yaml:"rotation_url"`
	RotationTTLSeconds int      `yaml:"rotation_ttl_seconds"`
	FailOpen           bool     `yaml:"fail_open"`
}

type Root struct {
	Env   string      `yaml:"env"`
	Proxy ProxyConfig `yaml:"proxy"`
	Local Config      `yaml:"local"`
	Dev   Config      `yaml:"dev"`
	Prod  Config      `yaml:"prod"`
}

type Config struct {
	Env string `yaml:"-"`

	Log struct {
		Level     string `yaml:"level"`
		Format    string `yaml:"format"`
		AddSource bool   `yaml:"add_source"`
	} `yaml:"log"`

	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`

	Kuper struct {
		BaseURL string `yaml:"base_url"`
		StoreID int    `yaml:"store_id"`
	} `yaml:"kuper"`

	CLI struct {
		CategoryID int    `yaml:"category_id"`
		OutputFile string `yaml:"output_file"`
	} `yaml:"cli"`

	Pagination struct {
		PerPage     int `yaml:"per_page"`
		OffersLimit int `yaml:"offers_limit"`
		MaxPages    int `yaml:"max_pages"`
	} `yaml:"pagination"`

	HTTP struct {
		TimeoutSeconds int `yaml:"timeout_seconds"`
		Retries        int `yaml:"retries"`
	} `yaml:"http"`

	Proxy ProxyConfig `yaml:"proxy"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var root Root
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, err
	}

	env := strings.TrimSpace(strings.ToLower(root.Env))
	if env == "" {
		env = "local"
	}

	var p Config
	switch env {
	case "local":
		p = root.Local
	case "dev":
		p = root.Dev
	case "prod":
		p = root.Prod
	default:
		return nil, fmt.Errorf("unknown env=%q (expected local|dev|prod)", env)
	}
	p.Env = env

	if isProxyEmpty(p.Proxy) && !isProxyEmpty(root.Proxy) {
		p.Proxy = root.Proxy
	}

	applyDefaults(&p)
	return &p, nil
}

func isProxyEmpty(px ProxyConfig) bool {
	return strings.TrimSpace(px.Mode) == "" && len(px.List) == 0 && strings.TrimSpace(px.RotationURL) == ""
}

func applyDefaults(p *Config) {
	if p.Kuper.BaseURL == "" {
		p.Kuper.BaseURL = "https://kuper.ru"
	}

	if p.Server.Host == "" {
		p.Server.Host = "0.0.0.0"
	}
	if p.Server.Port == 0 {
		p.Server.Port = 7891
	}

	if p.Pagination.PerPage <= 0 {
		p.Pagination.PerPage = 5
	}
	if p.Pagination.PerPage > 5 {
		p.Pagination.PerPage = 5
	}
	if p.Pagination.OffersLimit <= 0 {
		p.Pagination.OffersLimit = 10
	}
	if p.Pagination.MaxPages <= 0 {
		p.Pagination.MaxPages = 500
	}

	if p.HTTP.TimeoutSeconds <= 0 {
		p.HTTP.TimeoutSeconds = 30
	}
	if p.HTTP.Retries < 0 {
		p.HTTP.Retries = 0
	}

	if p.Log.Level == "" {
		if p.Env == "prod" {
			p.Log.Level = "info"
		} else {
			p.Log.Level = "debug"
		}
	}
	if p.Log.Format == "" {
		if p.Env == "prod" {
			p.Log.Format = "json"
		} else {
			p.Log.Format = "text"
		}
	}

	p.Proxy.Mode = strings.ToLower(strings.TrimSpace(p.Proxy.Mode))
	if p.Proxy.Mode == "" {
		p.Proxy.Mode = "disabled"
	}

	if len(p.Proxy.List) > 0 {
		clean := make([]string, 0, len(p.Proxy.List))
		for _, s := range p.Proxy.List {
			s = strings.TrimSpace(s)
			if s != "" {
				clean = append(clean, s)
			}
		}
		p.Proxy.List = clean
	}

	if p.Proxy.RotationTTLSeconds <= 0 {
		p.Proxy.RotationTTLSeconds = 10
	}
}
