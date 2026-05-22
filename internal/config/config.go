package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App       AppConfig
	Server    ServerConfig
	DB        DBConfig
	JWT       JWTConfig
	Storage   StorageConfig
	Payments  PaymentsConfig
	Orders    OrdersConfig
	Metrics   MetricsConfig
	Dev       DevConfig
	Sanctions SanctionsConfig
	Bootstrap BootstrapConfig
}

type AppConfig struct {
	Name        string
	Env         string
	LogLevel    string
	AutoMigrate bool
}

type ServerConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
}

type DBConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthTimeout   time.Duration
}

type JWTConfig struct {
	Issuer           string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration
	AccessSecretKey  string
	RefreshSecretKey string
}

type StorageConfig struct {
	Provider      string
	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	UseSSL        bool
	LocalPath     string
	PublicBaseURL string
}

type PaymentsConfig struct {
	Provider    string
	CallbackURL string
	PublicKey   string
	SecretKey   string
}

type OrdersConfig struct {
	PostingFee            float64
	ResponseSubmissionFee float64
	DefaultCurrency       string
}

type BootstrapConfig struct {
	EnableAdmin   bool
	AdminEmail    string
	AdminPassword string
}

type MetricsConfig struct {
	Enabled bool
	Path    string
	Public  bool
}

type DevConfig struct {
	EnablePaymentEndpoints bool
}

type SanctionsConfig struct {
	AutoAssignCourseOnLowRating bool
	DefaultLowRatingCourseID    string
}

func Load() (Config, error) {
	cfg := Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "buhpro-api"),
			Env:         getEnv("APP_ENV", "development"),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
			AutoMigrate: getEnvAsBool("AUTO_MIGRATE", true),
		},
		Server: ServerConfig{
			Host:            getEnv("HTTP_HOST", "0.0.0.0"),
			Port:            getEnv("HTTP_PORT", "8080"),
			ReadTimeout:     getEnvAsDuration("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getEnvAsDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getEnvAsDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
			AllowedOrigins:  getEnvAsSlice("HTTP_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		},
		DB: DBConfig{
			URL:             os.Getenv("DB_URL"),
			MaxConns:        int32(getEnvAsInt("DB_MAX_CONNS", 20)),
			MinConns:        int32(getEnvAsInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime: getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
			HealthTimeout:   getEnvAsDuration("DB_HEALTH_TIMEOUT", 2*time.Second),
		},
		JWT: JWTConfig{
			Issuer:           getEnv("JWT_ISSUER", "buhpro"),
			AccessTTL:        getEnvAsDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:       getEnvAsDuration("JWT_REFRESH_TTL", 720*time.Hour),
			AccessSecretKey:  os.Getenv("JWT_ACCESS_SECRET"),
			RefreshSecretKey: os.Getenv("JWT_REFRESH_SECRET"),
		},
		Storage: StorageConfig{
			Provider:      getEnv("STORAGE_PROVIDER", "local"),
			Endpoint:      getEnv("STORAGE_ENDPOINT", ""),
			Region:        getEnv("STORAGE_REGION", ""),
			Bucket:        getEnv("STORAGE_BUCKET", ""),
			AccessKey:     getEnv("STORAGE_ACCESS_KEY", ""),
			SecretKey:     getEnv("STORAGE_SECRET_KEY", ""),
			UseSSL:        getEnvAsBool("STORAGE_USE_SSL", true),
			LocalPath:     getEnv("STORAGE_LOCAL_PATH", "uploads"),
			PublicBaseURL: getEnv("STORAGE_PUBLIC_BASE_URL", "/uploads"),
		},
		Payments: PaymentsConfig{
			Provider:    getEnv("PAYMENTS_PROVIDER", "mock"),
			CallbackURL: getEnv("PAYMENTS_CALLBACK_URL", ""),
			PublicKey:   getEnv("PAYMENTS_PUBLIC_KEY", ""),
			SecretKey:   getEnv("PAYMENTS_SECRET_KEY", ""),
		},
		Orders: OrdersConfig{
			PostingFee:            getEnvAsFloat("ORDER_POSTING_FEE", 1000),
			ResponseSubmissionFee: getEnvAsFloat("RESPONSE_SUBMISSION_FEE", 500),
			DefaultCurrency:       strings.ToUpper(getEnv("DEFAULT_CURRENCY", "KZT")),
		},
		Metrics: MetricsConfig{
			Enabled: getEnvAsBool("METRICS_ENABLED", true),
			Path:    getEnv("METRICS_PATH", "/metrics"),
			Public:  getEnvAsBool("METRICS_PUBLIC", false),
		},
		Dev: DevConfig{
			EnablePaymentEndpoints: getEnvAsBool("ENABLE_DEV_PAYMENT_ENDPOINTS", false),
		},
		Sanctions: SanctionsConfig{
			AutoAssignCourseOnLowRating: getEnvAsBool("AUTO_ASSIGN_COURSE_ON_LOW_RATING", false),
			DefaultLowRatingCourseID:    strings.TrimSpace(getEnv("DEFAULT_LOW_RATING_COURSE_ID", "")),
		},
		Bootstrap: BootstrapConfig{
			EnableAdmin:   getEnvAsBool("BOOTSTRAP_ADMIN_ENABLED", false),
			AdminEmail:    getEnv("BOOTSTRAP_ADMIN_EMAIL", ""),
			AdminPassword: getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		},
	}

	missing := make([]string, 0, 3)
	if cfg.DB.URL == "" {
		missing = append(missing, "DB_URL")
	}
	if cfg.JWT.AccessSecretKey == "" {
		missing = append(missing, "JWT_ACCESS_SECRET")
	}
	if cfg.JWT.RefreshSecretKey == "" {
		missing = append(missing, "JWT_REFRESH_SECRET")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if cfg.Bootstrap.EnableAdmin {
		if cfg.Bootstrap.AdminEmail == "" || cfg.Bootstrap.AdminPassword == "" {
			return Config{}, fmt.Errorf("BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD are required when BOOTSTRAP_ADMIN_ENABLED=true")
		}
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return def
}

func getEnvAsBool(key string, def bool) bool {
	value := getEnv(key, strconv.FormatBool(def))
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvAsInt(key string, def int) int {
	value := getEnv(key, strconv.Itoa(def))
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvAsFloat(key string, def float64) float64 {
	value := getEnv(key, fmt.Sprintf("%v", def))
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvAsDuration(key string, def time.Duration) time.Duration {
	value := getEnv(key, def.String())
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvAsSlice(key string, def []string) []string {
	value := getEnv(key, "")
	if value == "" {
		return def
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return def
	}
	return result
}
