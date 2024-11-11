package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

// Configuration struct to hold all config values
type Configuration struct {
	Server struct {
		IP   string `yaml:"ip"`
		Port string `yaml:"port"`
	} `yaml:"server"`

	Postgres struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
	} `yaml:"postgres"`

	Redis struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`

	RabbitMQ struct {
		Username  string `yaml:"username"`
		Password  string `yaml:"password"`
		Host      string `yaml:"host"`
		Port      string `yaml:"port"`
		Exchanges struct {
			Currency      string `yaml:"currency"`
			User          string `yaml:"user"`
			Notifications string `yaml:"notifications"`
		} `yaml:"exchanges"`
	} `yaml:"rabbitmq"`

	JWT struct {
		SSL struct {
			Admin struct {
				PrivateKey []string `yaml:"private_key"`
				PublicKey  []string `yaml:"public_key"`
			} `yaml:"admin"`
			User struct {
				PrivateKey []string `yaml:"private_key"`
				PublicKey  []string `yaml:"public_key"`
			} `yaml:"user"`
		} `yaml:"ssl"`
		Issuer      string `yaml:"issuer"`
		AccessToken struct {
			Expiration int `yaml:"expiration"`
		} `yaml:"access_token"`
		RefreshToken struct {
			Expiration int `yaml:"expiration"`
		} `yaml:"refresh_token"`
	} `yaml:"jwt"`

	OTP struct {
		AUTH struct {
			Name       string `yaml:"name"` // Name of the OTP service
			TTL        int    `yaml:"ttl"`  // Time-to-live in seconds
			Length     int    `yaml:"length"`
			RetryLimit int    `yaml:"retry_limit"`
		} `yaml:"auth"`
	} `yaml:"otp"`
}

// Config holds the loaded configuration values
var Config Configuration

// LoadConfig loads configuration from config.yaml
func LoadConfig() {
	// Use os.ReadFile instead of ioutil.ReadFile
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// Unmarshal YAML data into the Config structure
	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}
}

// GetRabbitMQURL constructs the RabbitMQ connection URL from config values
func GetRabbitMQURL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		Config.RabbitMQ.Username,
		Config.RabbitMQ.Password,
		Config.RabbitMQ.Host,
		Config.RabbitMQ.Port,
	)
}
