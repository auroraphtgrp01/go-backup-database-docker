package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User đại diện cho một người dùng trong hệ thống
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"` // Không trả về password trong JSON
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HashPassword mã hóa mật khẩu người dùng
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash so sánh mật khẩu với hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Auth đại diện cho thông tin đăng nhập gửi lên từ client
type Auth struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// JWTClaims là các thông tin được mã hóa trong JWT token
type JWTClaims struct {
	Username string `json:"username"`
	UserID   int64  `json:"user_id"`
}
