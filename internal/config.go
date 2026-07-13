// Package internal contains all shared components across the application.
package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	DevelopmentEnv = "development"
	TestingEnv     = "testing"
	StagingEnv     = "staging"
	ProductionEnv  = "production"

	// envTrue is the canonical string value for boolean "true" env vars.
	envTrue = "true"
)

// StorageProvider defines the supported storage backends.
type StorageProvider string

const (
	// MockStorageProvider is the mock storage provider for testing.
	MockStorageProvider StorageProvider = "mock"

	// LocalFSProvider is the local filesystem storage provider.
	LocalFSProvider StorageProvider = "local"

	// S3Provider is the S3-compatible storage provider.
	S3Provider StorageProvider = "s3"
)

// S3Config holds the configuration for S3-compatible storage.
type S3Config struct {
	Endpoint   string `json:"endpoint"` // "localhost:9000" or "nyc3.digitaloceanspaces.com"
	Region     string `json:"region"`   // "us-east-1" or "nyc3"
	AccessKey  string `json:"-"`        // Never included in JSON (security)
	SecretKey  string `json:"-"`        // Never included in JSON (security)
	BucketName string `json:"bucket_name"`
	UseSSL     bool   `json:"use_ssl"`
}

// VLMConfig holds the configuration for the image-captioning VLM endpoint —
// a generic OpenAI-compatible vision chat-completions API (works against
// self-hosted vLLM/Ollama or any hosted open-weight-model provider using
// that shape).
type VLMConfig struct {
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	APIKey  string `json:"-"` // Never included in JSON (security); optional — self-hosted servers often need no auth
}

// EmailConfig groups transactional email configurations.
type EmailConfig struct {
	// Transactional holds configuration for user-triggered emails.
	Transactional TransactionalEmailConfig
}

// TransactionalEmailConfig holds configuration for user-triggered emails
// such as password reset, email verification, etc.
type TransactionalEmailConfig struct {
	// Provider is the email delivery backend to use for transactional emails.
	Provider Provider

	// FromName is the display name for the sender.
	FromName string

	// FromAddress is the email address for the sender.
	FromAddress string

	// SMTPHost is the SMTP server host for sending emails.
	SMTPHost string

	// SMTPPort is the SMTP server port for sending emails.
	SMTPPort int

	// SMTPUsername is the username for SMTP authentication.
	SMTPUsername string

	// SMTPPassword is the password for SMTP authentication.
	SMTPPassword string
}

type Config struct {
	Logger             *slog.Logger    `json:"-"`                     // Never included in JSON (not serializable)
	LogLevel           slog.Level      `json:"log_level"`             // Log level (debug, info, warn, error)
	Env                string          `json:"env"`                   // Environment (development, staging, production)
	JWTSecret          string          `json:"-"`                     // Never included in JSON (security)
	JWTExpiryHours     int             `json:"jwt_expiry_hours"`      // JWT token expiry in hours
	PostgresURI        string          `json:"-"`                     // Never included in JSON (security - contains credentials)
	APIHost            string          `json:"api_host"`              // API host (e.g., "localhost:8000")
	FrontendHost       string          `json:"frontend_host"`         // Frontend host (e.g., "localhost:3000")
	Port               int             `json:"port"`                  // Server port
	CORSAllowedOrigins []string        `json:"cors_allowed_origins"`  // CORS allowed origins
	CORSAllowedMethods []string        `json:"cors_allowed_methods"`  // CORS allowed methods
	StorageProvider    StorageProvider `json:"storage_provider"`      // Storage provider (local, s3)
	S3Config           *S3Config       `json:"s3_config"`             // S3 configuration
	EmailConfig        *EmailConfig    `json:"email_config"`          // Email configuration
	ApifyToken         string          `json:"APIFY_API_TOKEN"`       // Apify API token for triggering actors
	ApifyTikTokActorID string          `json:"APIFY_TIKTOK_ACTOR_ID"` // Apify Actor ID for TikTok media extraction
	ApifyIGActorID     string          `json:"APIFY_IG_ACTOR_ID"`     // Apify Actor ID for Instagram media extraction
	DoclingBaseURL     string          `json:"docling_base_url"`      // Base URL of the Docling Serve instance
	VLMConfig          *VLMConfig      `json:"vlm_config"`            // VLM configuration for image captioning

	// GoogleClientID is the OAuth 2.0 client ID for Google Sign-In.
	// Used server-side to verify Google ID tokens from the frontend.
	GoogleClientID string `json:"-"`

	// SentryDSN is the Data Source Name for Sentry error tracking.
	SentryDSN string `json:"-"`
}

// IsS3OK checks if the S3 configuration is valid.
func IsS3OK(c *S3Config) error {
	if c == nil {
		return errors.New("S3Config is not set")
	}
	if c.Endpoint == "" || c.AccessKey == "" || c.SecretKey == "" ||
		c.BucketName == "" {
		return errors.New("S3Config is incomplete")
	}
	return nil
}

// IsVLMOK checks if the VLM configuration is valid.
func IsVLMOK(c *VLMConfig) error {
	if c == nil {
		return errors.New("VLMConfig is not set")
	}
	if c.BaseURL == "" || c.Model == "" {
		return errors.New("VLMConfig is incomplete")
	}
	return nil
}

// IsOK checks if the configuration is valid.
func (c *Config) IsOK() error {
	if c.PostgresURI == "" {
		return errors.New("PostgresURI is not set")
	}
	if c.JWTSecret == "" {
		return errors.New("JWTSecret is not set")
	}
	if c.EmailConfig == nil {
		return errors.New("EmailConfig is not set")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("port is invalid")
	}
	if c.JWTExpiryHours <= 0 {
		return errors.New("JWTExpiryHours is invalid")
	}

	if c.ApifyToken == "" {
		return errors.New("APIFY_API_TOKEN is not set")
	}
	return nil
}

// IsProduction returns true if the environment is production.
func (c *Config) IsProduction() bool {
	// By default, empty ENV is assumed as production.
	// This is to prevent any unwanted data changes/migrations
	// that are intended for development and testing envs only.
	return c.Env == "production" || c.Env == ""
}

// IsTesting returns true if the environment is testing.
func (c *Config) IsTesting() bool {
	return c.Env == "testing"
}

// IsLocal returns true if the environment is development or testing.
func (c *Config) IsLocal() bool {
	return c.Env == "development" || c.Env == "testing"
}

// APISchemes returns the API schemes based on the APIHost.
func (c *Config) APISchemes() []string {
	schemes := []string{"http"}
	if !strings.Contains(c.APIHost, "localhost") {
		schemes[0] = "https"
	}
	return schemes
}

var defaultConfig *Config

// NewConfig creates a new Config instance by reading environment variables.
func NewConfig(envs ...string) *Config {
	if defaultConfig != nil {
		return defaultConfig
	}

	// Load .env file FIRST before reading any environment variables
	loadEnvFile(envs)

	// Parse basic configuration
	appLogLevelStr := getEnv("APP_LOG_LEVEL", "1")
	appLogLevel, err := strconv.Atoi(appLogLevelStr)
	if err != nil {
		appLogLevel = 1 // Default to Info level
	}
	appEnv := getEnv("APP_ENV", "development")
	logLevel := parseLogLevel(appLogLevel)
	logger := createLogger(appEnv, logLevel)

	// Parse hosts and database
	apiHost := getEnv("API_HOST", "localhost:8000")
	frontendHost := getEnv("FRONTEND_HOST", "localhost:5173")
	appDBURI := getEnv("APP_DB_URI", "")
	if appDBURI == "" {
		logger.Error("APP_DB_URI is not set! Exiting.")
	}

	// Parse JWT configuration
	appJwtSecret := getEnv("APP_JWT_SECRET", "")
	if appJwtSecret == "" {
		logger.Error("APP_JWT_SECRET is not set! Exiting.")
	}
	jwtExpiryHours := parseJWTExpiryHours(logger)

	// Parse server configuration
	port := parsePort(logger)
	corsAllowedOriginsStr := getEnv("CORS_ALLOWED_ORIGINS", "*")
	allowedMethods := parseCORSMethods()

	// Parse storage provider
	storageProvider := StorageProvider(getEnv("STORAGE_PROVIDER", "local"))

	// Load S3 and Email configuration
	s3Config := new(S3Config)
	if storageProvider == S3Provider {
		s3Config = loadS3ConfigOrDefault(logger, envs...)
	}
	emailConfig := loadEmailConfigOrDefault(logger, envs...)

	// Parse Google OAuth client ID
	googleClientID := getEnv("GOOGLE_CLIENT_ID", "")

	sentryDSN := getEnv("SENTRY_DSN", "")

	// Parse Apify configuration
	apifyToken := getEnv("APIFY_API_TOKEN", "")
	apifyTikTokActorID := getEnv("APIFY_TIKTOK_ACTOR_ID", "")
	apifyIGActorID := getEnv("APIFY_IG_ACTOR_ID", "")

	// Parse Docling configuration
	doclingBaseURL := getEnv("DOCLING_BASE_URL", "http://localhost:5001")

	// Load VLM configuration (image captioning) — no provider gate, unlike
	// S3: it's always relevant once the ingestion pipeline is active.
	vlmConfig := loadVLMConfigOrDefault(logger, envs...)

	defaultConfig = &Config{
		Logger:             logger,
		LogLevel:           logLevel,
		Env:                appEnv,
		JWTSecret:          appJwtSecret,
		JWTExpiryHours:     jwtExpiryHours,
		PostgresURI:        appDBURI,
		APIHost:            apiHost,
		FrontendHost:       frontendHost,
		Port:               port,
		CORSAllowedOrigins: strings.Split(corsAllowedOriginsStr, ","),
		CORSAllowedMethods: allowedMethods,
		S3Config:           s3Config,
		EmailConfig:        emailConfig,
		GoogleClientID:     googleClientID,
		SentryDSN:          sentryDSN,
		StorageProvider:    storageProvider,
		ApifyToken:         apifyToken,
		ApifyTikTokActorID: apifyTikTokActorID,
		ApifyIGActorID:     apifyIGActorID,
		DoclingBaseURL:     doclingBaseURL,
		VLMConfig:          vlmConfig,
	}
	return defaultConfig
}

// NewS3Config creates a new S3Config from environment variables.
func NewS3Config(envs ...string) (*S3Config, error) {
	// Note: .env file should already be loaded by NewConfig() before calling this.
	// This function only loads .env when called independently (e.g., in tests).
	if len(envs) > 0 && envs[0] != "" {
		source := envs[0]
		err := godotenv.Load(source)
		if err != nil {
			fmt.Printf("Warning: Failed to load .env file: %s\n", err.Error())
		} else {
			fmt.Printf("(NewS3Config) Loaded environment from: %s\n", source)
		}
	}

	endpoint := getEnv("S3_ENDPOINT", "")
	if endpoint == "" {
		return nil, errors.New("S3_ENDPOINT is not set")
	}
	region := getEnv("S3_REGION", "us-east-1")
	accessKey := getEnv("S3_ACCESS_KEY", "")
	if accessKey == "" {
		return nil, errors.New("S3_ACCESS_KEY is not set")
	}
	secretKey := getEnv("S3_SECRET", "")
	if secretKey == "" {
		return nil, errors.New("S3_SECRET is not set")
	}
	useSSLStr := getEnv("S3_USE_SSL", envTrue)
	bucketName := getEnv("S3_BUCKET_NAME", "")
	if bucketName == "" {
		return nil, errors.New("S3_BUCKET_NAME is not set")
	}
	return &S3Config{
		Endpoint:   endpoint,
		Region:     region,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		BucketName: bucketName,
		UseSSL:     useSSLStr == envTrue,
	}, nil
}

// NewVLMConfig creates a new VLMConfig from environment variables.
func NewVLMConfig(envs ...string) (*VLMConfig, error) {
	// Note: .env file should already be loaded by NewConfig() before calling this.
	// This function only loads .env when called independently (e.g., in tests).
	if len(envs) > 0 && envs[0] != "" {
		source := envs[0]
		if err := godotenv.Load(source); err != nil {
			fmt.Printf("Warning: Failed to load .env file: %s\n", err.Error())
		} else {
			fmt.Printf("(NewVLMConfig) Loaded environment from: %s\n", source)
		}
	}

	baseURL := getEnv("VLM_BASE_URL", "")
	if baseURL == "" {
		return nil, errors.New("VLM_BASE_URL is not set")
	}
	model := getEnv("VLM_MODEL", "")
	if model == "" {
		return nil, errors.New("VLM_MODEL is not set")
	}
	return &VLMConfig{
		BaseURL: baseURL,
		Model:   model,
		APIKey:  getEnv("VLM_API_KEY", ""),
	}, nil
}

// NewEmailConfig creates a new EmailConfig from environment variables.
func NewEmailConfig(envs ...string) (*EmailConfig, error) {
	// Note: .env file should already be loaded by NewConfig() before calling this.
	// This function only loads .env when called independently (e.g., in tests).
	if len(envs) > 0 && envs[0] != "" {
		source := envs[0]
		if err := godotenv.Load(source); err != nil {
			fmt.Printf("Warning: Failed to load .env file: %s\n", err.Error())
		} else {
			fmt.Printf("(NewEmailConfig) Loaded environment from: %s\n", source)
		}
	}

	txProvider := Provider(getEnv("EMAIL_PROVIDER_TRANSACTIONAL", "mock"))

	validProviders := map[Provider]bool{
		ProviderMock:    true,
		ProviderMailHog: true,
		ProviderSMTP:    true,
	}

	if !validProviders[txProvider] {
		return nil, fmt.Errorf("invalid EMAIL_PROVIDER_TRANSACTIONAL: %s", txProvider)
	}

	// Shared values
	fromName := getEnv("EMAIL_FROM_NAME", "Planet League")
	fromAddress := getEnv("EMAIL_FROM_ADDRESS", "noreply@planetleague.com")

	smtpHost := getEnv("SMTP_HOST", "localhost")
	smtpPortStr := getEnv("SMTP_PORT", "1025")
	smtpPort, err := strconv.Atoi(smtpPortStr)
	smtpUsername := getEnv("SMTP_USERNAME", "")
	smtpPassword := getEnv("SMTP_PASSWORD", "")
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %s", smtpPortStr)
	}

	return &EmailConfig{
		Transactional: TransactionalEmailConfig{
			Provider:     txProvider,
			FromName:     fromName,
			FromAddress:  fromAddress,
			SMTPHost:     smtpHost,
			SMTPPort:     smtpPort,
			SMTPUsername: smtpUsername,
			SMTPPassword: smtpPassword,
		},
	}, nil
}
