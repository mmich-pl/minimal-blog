package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload"
)

func LoadConfig() (*Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Config struct {
	S3         S3 `envPrefix:"S3_"`
	Scylla     Scylla
	HTTPServer HTTPServer
	Neo4j      Neo4j `envPrefix:"NEO4J_"`
	Redis      Redis `envPrefix:"REDIS_"`
}

type Redis struct {
	Address string        `json:"address" env:"ADDRESS" envDefault:"localhost:6379"`
	TTL     time.Duration `json:"timeout" env:"TTL" envDefault:"5m"`
}

type S3 struct {
	Key    string `env:"KEY" envDefault:"root"`
	Secret string `env:"SECRET" envDefault:"Secret1!"`
	Bucket string `env:"BUCKET" envDefault:"posts-storage"`
	Region string `env:"REGION" envDefault:"us-east-1"`

	Port    int    `env:"PORT" envDefault:"9000"`
	BaseUrl string `env:"BASE_URL" envDefault:"http://127.0.0.1"`
}

type Scylla struct {
	Host     string `env:"SCYLLA_HOST"`
	Keyspace string `env:"SCYLLA_KEYSPACE"`
}

type Neo4j struct {
	Host     string `env:"HOST" envDefault:"127.0.0.1"`
	Port     int    `env:"PORT" envDefault:"7687"`
	Username string `env:"USERNAME" envDefault:"neo4j"`
	Password string `env:"PASSWORD" envDefault:"Secret!1"`
}

type HTTPServer struct {
	IdleTimeout  time.Duration `env:"HTTP_SERVER_IDLE_TIMEOUT" envDefault:"60s"`
	Port         int           `env:"PORT" envDefault:"8080"`
	ReadTimeout  time.Duration `env:"HTTP_SERVER_READ_TIMEOUT" envDefault:"1s"`
	WriteTimeout time.Duration `env:"HTTP_SERVER_WRITE_TIMEOUT" envDefault:"2s"`
}
