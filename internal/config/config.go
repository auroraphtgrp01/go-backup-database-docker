package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config chứa các thông tin cấu hình từ file .env
type Config struct {
	DBUser             string
	DBPassword         string
	ContainerName      string
	DBName             string
	GoogleClientID     string
	GoogleClientSecret string
	FolderDrive        string
	CronSchedule       string
	BackupDir          string
	TokenDir           string
	WebAppPort         string
}

// LoadConfig nạp cấu hình từ file .env
func LoadConfig() (*Config, error) {
	// Nạp biến môi trường từ file .env
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	// Xác định đường dẫn thư mục gốc
	rootDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current directory: %w", err)
	}

	// Thiết lập đường dẫn các thư mục
	backupDir := filepath.Join(rootDir, "backups")
	tokenDir := filepath.Join(rootDir, "token")

	// Đảm bảo các thư mục tồn tại
	os.MkdirAll(backupDir, 0755)
	os.MkdirAll(tokenDir, 0755)

	// Lấy giá trị port hoặc sử dụng mặc định
	webAppPort := os.Getenv("WEBAPP_PORT")
	if webAppPort == "" {
		webAppPort = "8080"
	}

	// Lấy giá trị từ các biến môi trường
	config := &Config{
		DBUser:             os.Getenv("DB_USER"),
		DBPassword:         os.Getenv("DB_PASSWORD"),
		ContainerName:      os.Getenv("CONTAINER_NAME"),
		DBName:             os.Getenv("DB_NAME"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		FolderDrive:        os.Getenv("FOLDER_DRIVE"),
		CronSchedule:       os.Getenv("CRON_SCHEDULE"),
		BackupDir:          backupDir,
		TokenDir:           tokenDir,
		WebAppPort:         webAppPort,
	}

	// Kiểm tra các biến bắt buộc
	if config.DBUser == "" || config.DBPassword == "" || config.ContainerName == "" || config.DBName == "" {
		return nil, fmt.Errorf("missing required environment variables: DB_USER, DB_PASSWORD, CONTAINER_NAME, DB_NAME")
	}

	if config.GoogleClientID == "" || config.GoogleClientSecret == "" || config.FolderDrive == "" {
		return nil, fmt.Errorf("missing required Google Drive environment variables")
	}

	return config, nil
}
