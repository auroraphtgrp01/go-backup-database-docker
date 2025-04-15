package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/backup-cronjob/internal/config"
	"github.com/backup-cronjob/internal/database"
	"github.com/backup-cronjob/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var cfg *config.Config

// Init khởi tạo module xác thực
func Init(c *config.Config) {
	cfg = c
}

// GenerateJWT tạo JWT token cho người dùng đã xác thực
func GenerateJWT(user *models.User) (string, error) {
	// Thiết lập thời gian hết hạn (24 giờ)
	expirationTime := time.Now().Add(24 * time.Hour)

	// Tạo JWT claims
	claims := jwt.MapClaims{
		"username": user.Username,
		"user_id":  user.ID,
		"exp":      expirationTime.Unix(),
	}

	// Tạo token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Ký token với secret
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT kiểm tra và xác thực JWT token
func ValidateJWT(tokenString string) (*models.JWTClaims, error) {
	// Parse JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Kiểm tra thuật toán
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// Xác thực token và lấy claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Kiểm tra thời gian hết hạn
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return nil, errors.New("token has expired")
			}
		}

		// Lấy thông tin từ claims
		userID, _ := claims["user_id"].(float64)
		username, _ := claims["username"].(string)

		return &models.JWTClaims{
			Username: username,
			UserID:   int64(userID),
		}, nil
	}

	return nil, errors.New("invalid token")
}

// Middleware xác thực JWT
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Kiểm tra header có Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
			return
		}

		// Kiểm tra định dạng Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header format must be Bearer {token}"})
			return
		}

		// Xác thực token
		claims, err := ValidateJWT(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Lưu thông tin người dùng vào context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}

// AuthenticateUser xác thực người dùng với username và password
func AuthenticateUser(auth *models.Auth) (*models.User, error) {
	// Tìm người dùng theo username
	user, err := database.GetUserByUsername(auth.Username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	// Kiểm tra mật khẩu
	if !models.CheckPasswordHash(auth.Password, user.Password) {
		return nil, errors.New("invalid username or password")
	}

	return user, nil
}
