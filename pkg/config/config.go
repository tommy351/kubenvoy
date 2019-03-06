package config

import (
	"strings"
	"time"

	"github.com/ansel1/merry"
	"github.com/fatih/structs"
	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	Log        LogConfig        `mapstructure:"log"`
	Envoy      EnvoyConfig      `mapstructure:"envoy"`
}

type ServerConfig struct {
	Address string `mapstructure:"address"`
}

type KubernetesConfig struct {
	Namespace    string        `mapstructure:"namespace"`
	ResyncPeriod time.Duration `mapstructure:"resyncPeriod"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type EnvoyConfig struct {
	Node string `mapstructure:"node"`
}

func ReadConfig() (*Config, error) {
	var config Config
	v := viper.New()

	// Bind to environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	s := structs.New(&Config{
		Server: ServerConfig{
			Address: ":4000",
		},
		Kubernetes: KubernetesConfig{
			Namespace:    "default",
			ResyncPeriod: time.Second * 2,
		},
		Log: LogConfig{
			Level: "info",
		},
	})
	s.TagName = "mapstructure"

	if err := v.MergeConfigMap(s.Map()); err != nil {
		return nil, merry.Wrap(err)
	}

	if err := v.Unmarshal(&config); err != nil {
		return nil, merry.Wrap(err)
	}

	return &config, nil
}

func MustReadConfig() *Config {
	conf, err := ReadConfig()

	if err != nil {
		panic(err)
	}

	return conf
}
