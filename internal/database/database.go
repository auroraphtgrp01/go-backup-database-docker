package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/backup-cronjob/internal/config"
	"github.com/backup-cronjob/internal/models"
	_ "modernc.org/sqlite"
)

// DB là đối tượng database chung cho ứng dụng
var DB *sql.DB

// InitDB khởi tạo kết nối database
func InitDB(cfg *config.Config) error {
	var err error

	// Kết nối đến database SQLite
	DB, err = sql.Open("sqlite", cfg.SQLiteDBPath)
	if err != nil {
		return fmt.Errorf("error connecting to SQLite database: %w", err)
	}

	// Kiểm tra kết nối
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("error pinging SQLite database: %w", err)
	}

	// Tạo schema
	if err = createSchema(); err != nil {
		return fmt.Errorf("error creating database schema: %w", err)
	}

	// Kiểm tra và tạo tài khoản admin nếu chưa tồn tại
	if err = ensureAdminExists(cfg); err != nil {
		return fmt.Errorf("error ensuring admin user exists: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

// createSchema tạo cấu trúc cơ sở dữ liệu nếu chưa tồn tại
func createSchema() error {
	// Tạo bảng users
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	return err
}

// ensureAdminExists đảm bảo tài khoản admin tồn tại trong hệ thống
func ensureAdminExists(cfg *config.Config) error {
	// Kiểm tra xem admin đã tồn tại chưa
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", cfg.AdminUsername).Scan(&count)
	if err != nil {
		return err
	}

	// Nếu admin chưa tồn tại, tạo mới
	if count == 0 {
		// Hash mật khẩu
		hashedPassword, err := models.HashPassword(cfg.AdminPassword)
		if err != nil {
			return err
		}

		// Tạo admin
		now := time.Now()
		_, err = DB.Exec(
			"INSERT INTO users (username, password, created_at, updated_at) VALUES (?, ?, ?, ?)",
			cfg.AdminUsername, hashedPassword, now, now,
		)
		if err != nil {
			return err
		}

		log.Printf("Admin user '%s' created successfully", cfg.AdminUsername)
	}

	return nil
}

// GetUserByUsername lấy thông tin người dùng theo username
func GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := DB.QueryRow(
		"SELECT id, username, password, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// Close đóng kết nối đến database
func Close() {
	if DB != nil {
		DB.Close()
	}
}
