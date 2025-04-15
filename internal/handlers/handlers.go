package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/backup-cronjob/internal/auth"
	"github.com/backup-cronjob/internal/config"
	"github.com/backup-cronjob/internal/database"
	"github.com/backup-cronjob/internal/dbdump"
	"github.com/backup-cronjob/internal/drive"
	"github.com/backup-cronjob/internal/models"
	"github.com/gin-gonic/gin"
)

// Handler quản lý tất cả các xử lý HTTP
type Handler struct {
	Config         *config.Config
	DatabaseDumper *dbdump.DatabaseDumper
	DriveUploader  *drive.DriveUploader
}

// NewHandler tạo instance mới của Handler
func NewHandler(cfg *config.Config) *Handler {
	// Khởi tạo authentication
	auth.Init(cfg)

	// Khởi tạo database
	if err := database.InitDB(cfg); err != nil {
		panic(fmt.Sprintf("Failed to initialize database: %v", err))
	}

	return &Handler{
		Config:         cfg,
		DatabaseDumper: dbdump.NewDatabaseDumper(cfg),
		DriveUploader:  drive.NewDriveUploader(cfg),
	}
}

// OperationResult chứa kết quả của một thao tác
type OperationResult struct {
	Success bool
	Message string
}

// IndexHandler xử lý trang chủ
func (h *Handler) IndexHandler(c *gin.Context) {
	log.Printf("IndexHandler - Đang xử lý request từ %s", c.Request.RemoteAddr)

	// Kiểm tra các phương thức xác thực khác nhau
	cookieValue, _ := c.Cookie("logged_in")
	authToken, _ := c.Cookie("auth_token")
	authHeader := c.GetHeader("Authorization")

	log.Printf("Auth check: logged_in cookie=[%s], auth_token cookie exists=[%v], auth header exists=[%v]",
		cookieValue, authToken != "", authHeader != "")

	// Nếu có bất kỳ phương thức xác thực hợp lệ nào, cho phép truy cập
	if cookieValue == "true" || authToken != "" || authHeader != "" {
		log.Printf("✅ User authenticated via: %s", getAuthMethod(cookieValue, authToken, authHeader))

		// Nếu có auth token cookie nhưng không có logged_in cookie, đặt logged_in cookie
		if authToken != "" && cookieValue != "true" {
			log.Printf("Setting logged_in cookie from auth_token cookie")
			c.SetCookie("logged_in", "true", 3600*24*30, "/", "", false, false)
			c.SetSameSite(http.SameSiteLaxMode)
		}

		// Hiển thị trang chính
		// Kiểm tra đã xác thực Google Drive chưa
		isAuthenticated := h.DriveUploader.CheckAuth()

		// Hiển thị trang chủ với trạng thái xác thực Google Drive
		if !isAuthenticated {
			c.HTML(http.StatusOK, "index.html", gin.H{
				"NeedAuth": true,
			})
			return
		}

		// Lấy danh sách các file backup
		backups, err := models.GetAllBackups(h.Config.BackupDir)
		if err != nil {
			c.HTML(http.StatusOK, "index.html", gin.H{
				"Error": fmt.Sprintf("Không thể lấy danh sách backup: %v", err),
			})
			return
		}

		// Lấy kết quả thao tác từ session nếu có
		var lastOperation *OperationResult
		if flashes := c.Request.URL.Query().Get("success"); flashes != "" {
			lastOperation = &OperationResult{
				Success: flashes == "true",
				Message: c.Request.URL.Query().Get("message"),
			}
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Backups":       backups,
			"LastOperation": lastOperation,
		})
		return
	}

	// Chưa xác thực, chuyển hướng đến trang đăng nhập
	log.Printf("❌ User not authenticated, redirecting to login page")
	c.Redirect(http.StatusFound, "/login")
}

// getAuthMethod trả về phương thức xác thực được sử dụng
func getAuthMethod(cookieValue, authToken, authHeader string) string {
	methods := []string{}

	if cookieValue == "true" {
		methods = append(methods, "logged_in cookie")
	}

	if authToken != "" {
		methods = append(methods, "auth_token cookie")
	}

	if authHeader != "" {
		methods = append(methods, "Authorization header")
	}

	if len(methods) == 0 {
		return "unknown"
	}

	return strings.Join(methods, ", ")
}

// AuthHandler xử lý trang xác thực
func (h *Handler) AuthHandler(c *gin.Context) {
	// Tạo URL xác thực và chuyển hướng người dùng trực tiếp đến trang đăng nhập Google
	authURL := h.DriveUploader.GetAuthURL()
	c.Redirect(http.StatusFound, authURL)
}

// OAuthCallbackHandler xử lý callback từ Google OAuth2
func (h *Handler) OAuthCallbackHandler(c *gin.Context) {
	// Lấy mã xác thực từ query parameters
	code := c.Query("code")
	if code == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"Error": "Không nhận được mã xác thực từ Google. Vui lòng thử lại.",
		})
		return
	}

	// Đổi mã xác thực lấy token
	_, err := h.DriveUploader.ExchangeAuthCode(code)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"Error": fmt.Sprintf("Lỗi xác thực: %v", err),
		})
		return
	}

	// Hiển thị trang thành công, JavaScript sẽ tự động đóng cửa sổ này
	c.HTML(http.StatusOK, "auth_success.html", gin.H{
		"Message": "Xác thực Google Drive thành công! Cửa sổ này sẽ tự động đóng.",
	})
}

// DumpHandler xử lý yêu cầu dump database
func (h *Handler) DumpHandler(c *gin.Context) {
	// Kiểm tra xác thực JWT từ local storage
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.Redirect(http.StatusSeeOther, "/?success=false&message=Vui lòng đăng nhập để thực hiện thao tác này")
		return
	}

	// Thực hiện dump database
	result, err := h.DatabaseDumper.DumpDatabase()
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?success=false&message="+fmt.Sprintf("Lỗi khi dump database: %v", err))
		return
	}

	c.Redirect(http.StatusSeeOther, "/?success=true&message="+fmt.Sprintf("Đã dump database thành công. File: %s", filepath.Base(result.FilePath)))
}

// UploadLastHandler xử lý yêu cầu upload file mới nhất
func (h *Handler) UploadLastHandler(c *gin.Context) {
	// Kiểm tra xác thực JWT
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.Redirect(http.StatusSeeOther, "/?success=false&message=Vui lòng đăng nhập để thực hiện thao tác này")
		return
	}

	// Kiểm tra xác thực Google Drive
	if !h.DriveUploader.CheckAuth() {
		c.Redirect(http.StatusSeeOther, "/auth")
		return
	}

	// Tìm file backup mới nhất
	latestBackup, err := models.FindLatestBackup(h.Config.BackupDir)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?success=false&message="+fmt.Sprintf("Không tìm thấy file backup: %v", err))
		return
	}

	// Upload file lên Drive
	err = h.DriveUploader.UploadFile(latestBackup.Path)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?success=false&message="+fmt.Sprintf("Lỗi khi upload file: %v", err))
		return
	}

	c.Redirect(http.StatusSeeOther, "/?success=true&message="+fmt.Sprintf("Đã upload file %s lên Google Drive", latestBackup.Name))
}

// UploadAllHandler xử lý yêu cầu upload tất cả file
func (h *Handler) UploadAllHandler(c *gin.Context) {
	// Kiểm tra xác thực JWT
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.Redirect(http.StatusSeeOther, "/?success=false&message=Vui lòng đăng nhập để thực hiện thao tác này")
		return
	}

	// Kiểm tra xác thực Google Drive
	if !h.DriveUploader.CheckAuth() {
		c.Redirect(http.StatusSeeOther, "/auth")
		return
	}

	// Upload tất cả file backup
	err := h.DriveUploader.UploadAllBackups()
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?success=false&message="+fmt.Sprintf("Lỗi khi upload tất cả file: %v", err))
		return
	}

	c.Redirect(http.StatusSeeOther, "/?success=true&message=Đã upload tất cả file backup lên Google Drive")
}

// UploadSingleHandler xử lý yêu cầu upload một file cụ thể
func (h *Handler) UploadSingleHandler(c *gin.Context) {
	// Kiểm tra xác thực JWT
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.Redirect(http.StatusSeeOther, "/?success=false&message=Vui lòng đăng nhập để thực hiện thao tác này")
		return
	}

	// Kiểm tra xác thực Google Drive
	if !h.DriveUploader.CheckAuth() {
		c.Redirect(http.StatusSeeOther, "/auth")
		return
	}

	fileID := c.Param("id")
	backups, err := models.GetAllBackups(h.Config.BackupDir)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?success=false&message="+fmt.Sprintf("Không thể lấy danh sách backup: %v", err))
		return
	}

	// Tìm file backup theo ID
	var targetBackup *models.BackupFile
	for _, backup := range backups {
		if backup.ID == fileID {
			targetBackup = backup
			break
		}
	}

	if targetBackup == nil {
		c.Redirect(http.StatusSeeOther, "/?success=false&message="+fmt.Sprintf("Không tìm thấy file backup có ID: %s", fileID))
		return
	}

	// Upload file lên Drive
	err = h.DriveUploader.UploadFile(targetBackup.Path)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?success=false&message="+fmt.Sprintf("Lỗi khi upload file: %v", err))
		return
	}

	c.Redirect(http.StatusSeeOther, "/?success=true&message="+fmt.Sprintf("Đã upload file %s lên Google Drive", targetBackup.Name))
}

// DownloadHandler xử lý yêu cầu tải xuống file backup
func (h *Handler) DownloadHandler(c *gin.Context) {
	// Kiểm tra xác thực JWT từ query parameter (từ client javascript)
	token := c.Query("token")
	if token == "" {
		c.Redirect(http.StatusSeeOther, "/?success=false&message=Vui lòng đăng nhập để thực hiện thao tác này")
		return
	}

	// Xác thực token
	_, err := auth.ValidateJWT(token)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?success=false&message=Phiên đăng nhập không hợp lệ hoặc đã hết hạn")
		return
	}

	fileID := c.Param("id")
	backups, err := models.GetAllBackups(h.Config.BackupDir)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?success=false&message="+fmt.Sprintf("Không thể lấy danh sách backup: %v", err))
		return
	}

	// Tìm file backup theo ID
	var targetBackup *models.BackupFile
	for _, backup := range backups {
		if backup.ID == fileID {
			targetBackup = backup
			break
		}
	}

	if targetBackup == nil {
		c.Redirect(http.StatusSeeOther, "/?success=false&message="+fmt.Sprintf("Không tìm thấy file backup có ID: %s", fileID))
		return
	}

	// Kiểm tra file có tồn tại không
	if _, err := os.Stat(targetBackup.Path); os.IsNotExist(err) {
		c.Redirect(http.StatusSeeOther, "/?success=false&message="+fmt.Sprintf("File %s không tồn tại", targetBackup.Name))
		return
	}

	// Trả về file để tải xuống
	c.FileAttachment(targetBackup.Path, targetBackup.Name)
}
