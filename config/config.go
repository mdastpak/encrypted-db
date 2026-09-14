package config

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

var configEmbed embed.FS

var Config Configuration

var configPath string = "config/config.yaml"
var migrationsPath string = "file://internal/db/migrations"

func SetConfigPath(path string) {
	configPath = path
}

func SetMigrationsPath(path string) {
	migrationsPath = path
}

func ConfigPath() string {
	return configPath
}

func MigrationsPath() string {
	return migrationsPath
}

type Configuration struct {
	Server struct {
		IP   string `yaml:"ip"`
		Port string `yaml:"port"`
	} `yaml:"server"`

	Postgres struct {
		Host            string `yaml:"host"`
		Port            string `yaml:"port"`
		User            string `yaml:"user"`
		Password        string `yaml:"password"`
		DBName          string `yaml:"dbname"`
		SSLMode         string `yaml:"sslmode"`
		MaxOpenConns    int    `yaml:"max_open_conns"`
		MaxIdleConns    int    `yaml:"max_idle_conns"`
		ConnMaxLifetime int    `yaml:"conn_max_lifetime_minutes"`
		ConnMaxIdleTime int    `yaml:"conn_max_idle_time_minutes"`
	} `yaml:"postgres"`

	Redis struct {
		Host         string `yaml:"host"`
		Port         string `yaml:"port"`
		Password     string `yaml:"password"`
		DB           int    `yaml:"db"`
		PoolSize     int    `yaml:"pool_size"`
		MinIdleConns int    `yaml:"min_idle_conns"`
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
			Name       string `yaml:"name"`
			TTL        int    `yaml:"ttl"`
			Length     int    `yaml:"length"`
			RetryLimit int    `yaml:"retry_limit"`
		} `yaml:"auth"`
	} `yaml:"otp"`

	WebSocket struct {
		AllowedOrigins []string `yaml:"allowed_origins"`
	} `yaml:"websocket"`
}

func LoadConfig() {
	var data []byte
	var err error

	if _, statErr := os.Stat(configPath); statErr == nil {
		data, err = os.ReadFile(configPath)
		if err != nil {
			log.Fatalf("Error reading config file: %v", err)
		}
	} else {
		data, err = configEmbed.ReadFile("config.example.yaml")
		if err != nil {
			log.Fatalf("Error reading embedded config: %v", err)
		}
	}

	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	setDefaults()
	overlayEnvVars()
	validateConfig()
}

func setDefaults() {
	if Config.Postgres.MaxOpenConns == 0 {
		Config.Postgres.MaxOpenConns = 25
	}
	if Config.Postgres.MaxIdleConns == 0 {
		Config.Postgres.MaxIdleConns = 5
	}
	if Config.Postgres.ConnMaxLifetime == 0 {
		Config.Postgres.ConnMaxLifetime = 5
	}
	if Config.Postgres.ConnMaxIdleTime == 0 {
		Config.Postgres.ConnMaxIdleTime = 5
	}

	if Config.Redis.PoolSize == 0 {
		Config.Redis.PoolSize = 10
	}
	if Config.Redis.MinIdleConns == 0 {
		Config.Redis.MinIdleConns = 2
	}

	if Config.Postgres.SSLMode == "" {
		Config.Postgres.SSLMode = "disable"
	}
}

func overlayEnvVars() {
	if v := os.Getenv("POSTGRES_PASSWORD"); v != "" {
		Config.Postgres.Password = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		Config.Redis.Password = v
	}
	if v := os.Getenv("RABBITMQ_USER"); v != "" {
		Config.RabbitMQ.Username = v
	}
	if v := os.Getenv("RABBITMQ_PASSWORD"); v != "" {
		Config.RabbitMQ.Password = v
	}
}

func validateConfig() {
	if Config.Postgres.Password == "" {
		log.Println("WARNING: POSTGRES_PASSWORD is not set")
	}
	if Config.RabbitMQ.Username == "" || Config.RabbitMQ.Password == "" {
		log.Println("WARNING: RABBITMQ credentials are not set")
	}
	if len(Config.JWT.SSL.User.PrivateKey) == 0 || len(Config.JWT.SSL.User.PublicKey) == 0 {
		log.Println("WARNING: JWT SSL keys not configured")
	}
}

func (c *Configuration) JWTPrivateKeyPath() string {
	return filepath.Join(c.JWT.SSL.User.PrivateKey...)
}

func (c *Configuration) JWTPublicKeyPath() string {
	return filepath.Join(c.JWT.SSL.User.PublicKey...)
}

func GetRabbitMQURL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		Config.RabbitMQ.Username,
		Config.RabbitMQ.Password,
		Config.RabbitMQ.Host,
		Config.RabbitMQ.Port,
	)
}
