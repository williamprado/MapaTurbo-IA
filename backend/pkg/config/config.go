package config

import (
	"log"
	"os"
	"reflect"

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
	viper.SetConfigType("env")

	// Read an optional .env file when it exists (local development).
	// In container/Swarm deployments there is usually no .env file and all
	// configuration comes from OS environment variables instead.
	envFile := path + "/.env"
	if _, err := os.Stat(envFile); err == nil {
		viper.SetConfigFile(envFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Warning: error reading .env file: %v", err)
		}
	}

	viper.AutomaticEnv()

	// viper.AutomaticEnv() only feeds viper.Get(); it does NOT make Unmarshal
	// pick up env-only keys. Without a .env file viper knows no keys, so
	// Unmarshal would return an empty struct even though the variables exist
	// in the environment. Explicitly bind every struct key so OS environment
	// variables are always honored (e.g. Docker/Swarm/Portainer deployments).
	bindEnvVars(reflect.TypeOf(Config{}))

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

// bindEnvVars binds each `mapstructure` tag of the given struct type to its
// matching environment variable, so viper.Unmarshal honors OS env vars even
// when no config file is present.
func bindEnvVars(t reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		if tag := t.Field(i).Tag.Get("mapstructure"); tag != "" {
			_ = viper.BindEnv(tag)
		}
	}
}
