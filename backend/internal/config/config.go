package config

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:",squash"`
	Database DatabaseConfig `mapstructure:",squash"`
	Redis    RedisConfig    `mapstructure:",squash"`
	Aliyun   AliyunConfig   `mapstructure:",squash"`
	OSS      OSSConfig      `mapstructure:",squash"`
}

type ServerConfig struct {
	Port                      string `mapstructure:"SERVER_PORT"`
	Mode                      string `mapstructure:"SERVER_MODE"`
	InternalSecret            string `mapstructure:"INTERNAL_API_SECRET"`
	STPWorkerPath             string `mapstructure:"STP_WORKER_PATH"`
	STPWorkerPython           string `mapstructure:"STP_WORKER_PYTHON"`
	DownloadNASDir            string `mapstructure:"DOWNLOAD_NAS_DIR"`
	DownloadNASDepartmentDirs string `mapstructure:"DOWNLOAD_NAS_DEPARTMENT_DIRS"`
}

type DatabaseConfig struct {
	Type     string `mapstructure:"DB_TYPE"`
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	DBName   string `mapstructure:"DB_NAME"`
	SSLMode  string `mapstructure:"DB_SSLMODE"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"REDIS_ADDR"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

type AliyunConfig struct {
	AccessKeyID     string `mapstructure:"ALIBABA_CLOUD_ACCESS_KEY_ID"`
	AccessKeySecret string `mapstructure:"ALIBABA_CLOUD_ACCESS_KEY_SECRET"`
}

type OSSConfig struct {
	Bucket   string `mapstructure:"OSS_BUCKET"`
	Endpoint string `mapstructure:"OSS_ENDPOINT"`
	Domain   string `mapstructure:"OSS_DOMAIN"`
}

var AppConfig Config

func LoadConfig() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv() // 读取环境变量

	// 允许环境变量覆盖
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 显式绑定环境变量（mapstructure 只影响 .env 文件，不影响环境变量）
	_ = viper.BindEnv("Server.STPWorkerPath", "STP_WORKER_PATH")
	_ = viper.BindEnv("Server.STPWorkerPython", "STP_WORKER_PYTHON")
	_ = viper.BindEnv("Server.InternalSecret", "INTERNAL_API_SECRET")
	_ = viper.BindEnv("Server.DownloadNASDir", "DOWNLOAD_NAS_DIR")
	_ = viper.BindEnv("Server.DownloadNASDepartmentDirs", "DOWNLOAD_NAS_DEPARTMENT_DIRS")
	_ = viper.BindEnv("Aliyun.AccessKeyID", "ALIBABA_CLOUD_ACCESS_KEY_ID")
	_ = viper.BindEnv("Aliyun.AccessKeySecret", "ALIBABA_CLOUD_ACCESS_KEY_SECRET")
	_ = viper.BindEnv("OSS.Bucket", "OSS_BUCKET")
	_ = viper.BindEnv("OSS.Endpoint", "OSS_ENDPOINT")
	_ = viper.BindEnv("OSS.Domain", "OSS_DOMAIN")

	// Debug: 检查环境变量是否被读取
	log.Printf("[CONFIG] AI_AGENT_PYTHON env=%q, STP_WORKER_PYTHON env=%q",
		os.Getenv("AI_AGENT_PYTHON"), os.Getenv("STP_WORKER_PYTHON"))
	log.Printf("[CONFIG] OSS config - BUCKET=%q, ENDPOINT=%q, ACCESS_KEY_ID=%q",
		os.Getenv("OSS_BUCKET"), os.Getenv("OSS_ENDPOINT"), os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID"))

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: .env file not found, using system environment variables")
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	// 将阿里云配置设置到系统环境变量（供阿里云 SDK 使用）
	if AppConfig.Aliyun.AccessKeyID != "" {
		os.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", AppConfig.Aliyun.AccessKeyID)
	}
	if AppConfig.Aliyun.AccessKeySecret != "" {
		os.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", AppConfig.Aliyun.AccessKeySecret)
	}

	// 将 OSS 配置设置到系统环境变量（供 OSS SDK 使用）
	if AppConfig.OSS.Bucket != "" {
		os.Setenv("OSS_BUCKET", AppConfig.OSS.Bucket)
	}
	if AppConfig.OSS.Endpoint != "" {
		os.Setenv("OSS_ENDPOINT", AppConfig.OSS.Endpoint)
	}
	if AppConfig.OSS.Domain != "" {
		os.Setenv("OSS_DOMAIN", AppConfig.OSS.Domain)
	}
	if AppConfig.Server.DownloadNASDir != "" {
		os.Setenv("DOWNLOAD_NAS_DIR", AppConfig.Server.DownloadNASDir)
	}
	if AppConfig.Server.DownloadNASDepartmentDirs != "" {
		os.Setenv("DOWNLOAD_NAS_DEPARTMENT_DIRS", AppConfig.Server.DownloadNASDepartmentDirs)
	}
}

// GetEnv returns the value of the environment variable named by the key.
// If the variable is not set, it returns the defaultValue.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
