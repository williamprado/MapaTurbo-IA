package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	NodeEnv                string `mapstructure:"NODE_ENV"`
	AppEnv                 string `mapstructure:"APP_ENV"`
	AppURL                 string `mapstructure:"APP_URL"`
	APIURL                 string `mapstructure:"API_URL"`
	ServerPort             string `mapstructure:"SERVER_PORT"`
	DatabaseURL            string `mapstructure:"DATABASE_URL"`
	RedisAddr              string `mapstructure:"REDIS_ADDR"`
	RedisPassword          string `mapstructure:"REDIS_PASSWORD"`
	RedisDB                int    `mapstructure:"REDIS_DB"`
	JWTSecret              string `mapstructure:"JWT_SECRET"`
	JWTExpiresIn           string `mapstructure:"JWT_EXPIRES_IN"`
	EncryptionKey          string `mapstructure:"ENCRYPTION_KEY"`
	BootstrapAdminEmail    string `mapstructure:"BOOTSTRAP_ADMIN_EMAIL"`
	BootstrapAdminPassword string `mapstructure:"BOOTSTRAP_ADMIN_PASSWORD"`
	MinioEndpoint          string `mapstructure:"MINIO_ENDPOINT"`
	MinioAccessKey         string `mapstructure:"MINIO_ACCESS_KEY"`
	MinioSecretKey         string `mapstructure:"MINIO_SECRET_KEY"`
	MinioBucket            string `mapstructure:"MINIO_BUCKET"`
	MinioUseSSL            bool   `mapstructure:"MINIO_USE_SSL"`
	DefaultCurrency        string `mapstructure:"DEFAULT_CURRENCY"`
	OpenAIApiKey           string `mapstructure:"OPENAI_API_KEY"`
	GeminiApiKey           string `mapstructure:"GEMINI_API_KEY"`
	XaiApiKey              string `mapstructure:"XAI_API_KEY"`
}

func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("")
	viper.SetConfigType("env")

	// Check if a specific file name config is needed, e.g., if we read .env
	// We check for .env in the path
	envFile := path + "/.env"
	if _, err := os.Stat(envFile); err == nil {
		viper.SetConfigFile(envFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Warning: error reading .env file: %v", err)
		}
	}

	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Defaults and fallbacks
	if config.ServerPort == "" {
		config.ServerPort = "8080"
	}
	if config.DefaultCurrency == "" {
		config.DefaultCurrency = "BRL"
	}

	return &config, nil
}
