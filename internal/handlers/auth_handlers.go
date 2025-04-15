package handlers

import (
	"net/http"

	"log"

	"github.com/backup-cronjob/internal/auth"
	"github.com/backup-cronjob/internal/models"
	"github.com/gin-gonic/gin"
)

// LoginPageHandler hiển thị trang đăng nhập
func (h *Handler) LoginPageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

// LoginHandler xử lý đăng nhập và trả về JWT token
func (h *Handler) LoginHandler(c *gin.Context) {
	var loginData models.Auth

	// Parse dữ liệu đăng nhập từ JSON
	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Xác thực người dùng
	user, err := auth.AuthenticateUser(&loginData)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Tạo JWT token
	token, err := auth.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Đặt cookie xác thực - đây là cách chính để xác thực
	// Dùng SameSite=None để đảm bảo cookie được gửi trong mọi trường hợp
	c.SetCookie(
		"logged_in", // Tên cookie
		"true",      // Giá trị
		3600*24*30,  // Thời gian sống (30 ngày)
		"/",         // Path
		"",          // Domain (empty = current domain)
		false,       // Secure (false trên HTTP, true trên HTTPS)
		false,       // HttpOnly - false để JavaScript có thể đọc cookie
	)

	// Đặt cookie thứ hai chứa JWT để đảm bảo phiên đăng nhập được duy trì ngay cả khi localStorage bị xóa
	c.SetCookie(
		"auth_token", // Tên cookie
		token,        // JWT Token
		3600*24*30,   // Thời gian sống (30 ngày)
		"/",          // Path
		"",           // Domain
		false,        // Secure
		true,         // HttpOnly - true để bảo vệ token khỏi JS
	)

	// Đảm bảo cookie có thuộc tính SameSite đúng
	c.SetSameSite(http.SameSiteLaxMode)

	// In log thông báo về cookie đã được đặt
	log.Printf("Login successful for user %s - Set cookies with expiry 30 days, SameSite=Lax", user.Username)

	// Trả về token để lưu trong localStorage (dự phòng)
	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"message": "Login successful",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// MeHandler trả về thông tin người dùng hiện tại
func (h *Handler) MeHandler(c *gin.Context) {
	// Lấy thông tin người dùng đã được lưu trong middleware
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Trả về thông tin người dùng
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       userID,
			"username": username,
		},
	})
}

// LogoutHandler xử lý đăng xuất
func (h *Handler) LogoutHandler(c *gin.Context) {
	// Lấy thông tin về người dùng đang đăng xuất để ghi log
	authHeader := c.GetHeader("Authorization")
	var username string
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token := authHeader[7:]
		claims, err := auth.ValidateJWT(token)
		if err == nil && claims != nil {
			username = claims.Username
		}
	}

	// Xóa cả hai cookie đăng nhập với cùng các tham số như khi tạo
	// Xóa cookie logged_in
	c.SetCookie(
		"logged_in", // Tên cookie
		"",          // Giá trị rỗng
		-1,          // Thời gian âm = xóa cookie
		"/",         // Path
		"",          // Domain
		false,       // Secure
		false,       // HttpOnly
	)

	// Xóa cookie auth_token
	c.SetCookie(
		"auth_token", // Tên cookie
		"",           // Giá trị rỗng
		-1,           // Thời gian âm = xóa cookie
		"/",          // Path
		"",           // Domain
		false,        // Secure
		true,         // HttpOnly
	)

	// Đảm bảo thuộc tính SameSite khớp với khi tạo cookie
	c.SetSameSite(http.SameSiteLaxMode)

	// Ghi log đăng xuất
	log.Printf("User %s logged out, all cookies deleted", username)

	// Phản hồi thành công
	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}
