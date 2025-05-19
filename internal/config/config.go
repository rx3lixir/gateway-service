package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Константы для ключей конфигурации
const (
	envKey               = "service_params.env"
	secretKey            = "secret_key.env"
	gateway_http_port    = "gateway_http_port.env"
	user_client_address  = "user_client_addr.env"
	auth_client_address  = "auth_client_addr.env"
	event_client_address = "event_client_addr.env"
)

// AppConfig представляет конфигурацию всего приложения
type AppConfig struct {
	Service ServiceParams `mapstructure:"service_params" validate:"required"`
	Server  ServerParams  `mapstructure:"server_params" validate:"required"`
	Clients ClientsParams `mapstructure:"clients_params" validate:"required"`
}

// ApplicationParams содержит общие параметры приложения
type ServiceParams struct {
	Env       string `mapstructure:"env" validate:"required,oneof=dev prod test"`
	SecretKey string `mapstructure:"secret_key" validate:"required"`
}

type ServerParams struct {
	HTTPPort string `mapstructure:"http_port" validate:"required"`
}

type ClientsParams struct {
	EventClientAddress string `mapstructure:"event_client_address" validate:"required"`
	AuthClientAddress  string `mapstructure:"auth_client_address" validate:"required"`
	UserClientAddress  string `mapstructure:"user_client_address" validate:"required"`
}

// EnvBindings возвращает мапу ключей конфигурации и соответствующих им переменных окружения
func envBindings() map[string]string {
	return map[string]string{
		envKey:               "SERVICE_KEY",
		secretKey:            "SECRET_KEY",
		gateway_http_port:    "GATEWAY_HTTP_PORT",
		event_client_address: "EVENT_CLIENT_ADDR",
		auth_client_address:  "AUTH_CLIENT_ADDR",
		user_client_address:  "USER_CLIENT_ADDR",
	}
}

// New загружает конфигурацию из файла и переменных окружения
func New() (*AppConfig, error) {
	v := viper.New()

	// Получаем рабочую директорию
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить рабочую директорию: %w", err)
	}

	v.AddConfigPath(filepath.Join(cwd, "internal", "config"))
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	// Привязка переменных окружения
	for configKey, envVar := range envBindings() {
		if err := v.BindEnv(configKey, envVar); err != nil {
			return nil, fmt.Errorf("ошибка привязки переменной окружения %s: %w", envVar, err)
		}
	}

	// Чтение конфигурации
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("ошибка чтения конфигурационного файла: %w", err)
	}

	var config AppConfig

	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("ошибка при декодировании конфигурации: %w", err)
	}

	// Валидация конфигурации
	validate := validator.New()

	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("ошибка валидации конфигурации: %w", err)
	}

	return &config, nil
}
