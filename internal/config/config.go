package config

import (
	"log/slog"
	reflect "reflect"

	"github.com/artarts36/specw"
	"github.com/caarlos0/env/v11"
)

// Config describes the application configuration.
type Config struct {
	// HTTPAddr is the address where the HTTP server listens.
	HTTPAddr string `env:"HTTP_ADDR" envDefault:":8080"`

	Target struct {
		URL specw.URL `env:"URL,required,notEmpty"`
	} `envPrefix:"TARGET_"`

	// Telegram is the configuration related to working with Telegram.
	Telegram TelegramConfig `envPrefix:"TELEGRAM_"`

	// IPHeaders defines which headers are used to determine the client IP address.
	IPHeaders []string `env:"IP_HEADERS"`

	Log struct {
		Level slog.Level `env:"LEVEL"`
	} `envPrefix:"LOG_"`
}

// TelegramConfig contains all environment variables related to Telegram.
type TelegramConfig struct {
	// CIDRURL is the url with the list of Telegram CIDRs.
	CIDRURL string `env:"CIDR_URL" envDefault:"https://core.telegram.org/resources/cidr.txt"`

	// CIDRUpdateInterval is how often (in seconds) to refresh the CIDR list.
	// We use int so it can be read from env conveniently.
	CIDRUpdateInterval specw.Duration `env:"CIDR_UPDATE_INTERVAL" envDefault:"24h"`
}

// Load loads configuration from environment variables.
func Load() (Config, error) {
	var cfg Config
	if err := env.ParseWithOptions(&cfg, env.Options{
		Prefix: "TWG_",
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf(specw.Duration{}): func(s string) (interface{}, error) {
				dur := specw.Duration{}
				if err := dur.UnmarshalString(s); err != nil {
					return nil, err
				}
				return dur, nil
			},
			reflect.TypeOf(specw.URL{}): func(s string) (interface{}, error) {
				u := specw.URL{}
				if err := u.UnmarshalString(s); err != nil {
					return nil, err
				}
				return u, nil
			},
		},
	}); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
