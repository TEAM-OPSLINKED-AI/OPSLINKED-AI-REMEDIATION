package config

import (
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	Log    LogConfig    `mapstructure:"log"`
	Server ServerConfig `mapstructure:"server"`
	SMTP   SMTPConfig   `mapstructure:"smtp"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type ServerConfig struct {
	Port            int `mapstructure:"port"`
	ShutdownTimeout int `mapstructure:"shutdownTimeout"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	FromEmail string `mapstructure:"fromEmail"`
	ToEmail   string `mapstructure:"toEmail"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.SetDefault("log.level", "info")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.shutdownTimeout", 15)
	viper.SetDefault("smtp.port", 587)

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err!= nil {
		if _, ok := err.(viper.ConfigFileNotFoundError);!ok {
			return
		}
	}

	err = viper.Unmarshal(&config)
	return
}