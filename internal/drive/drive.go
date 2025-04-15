package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/backup-cronjob/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// DriveUploader quản lý việc upload file lên Google Drive
type DriveUploader struct {
	Config *config.Config
}

// NewDriveUploader tạo instance mới của DriveUploader
func NewDriveUploader(cfg *config.Config) *DriveUploader {
	return &DriveUploader{
		Config: cfg,
	}
}

// GetOAuthConfig trả về cấu hình OAuth2
func (d *DriveUploader) GetOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     d.Config.GoogleClientID,
		ClientSecret: d.Config.GoogleClientSecret,
		Scopes:       []string{drive.DriveFileScope},
		RedirectURL:  fmt.Sprintf("http://localhost:%s/callback", d.Config.WebAppPort),
		Endpoint:     google.Endpoint,
	}
}

// GetAuthURL tạo URL xác thực
func (d *DriveUploader) GetAuthURL() string {
	config := d.GetOAuthConfig()
	return config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeAuthCode đổi mã xác thực lấy token
func (d *DriveUploader) ExchangeAuthCode(code string) (*oauth2.Token, error) {
	config := d.GetOAuthConfig()
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("không thể đổi mã xác thực: %v", err)
	}

	// Lưu token
	tokenFile := filepath.Join(d.Config.TokenDir, "token.json")
	err = d.saveToken(tokenFile, token)
	if err != nil {
		return nil, fmt.Errorf("không thể lưu token: %v", err)
	}

	return token, nil
}

// getClient lấy OAuth2 client để truy cập Google Drive API
func (d *DriveUploader) getClient() (*drive.Service, error) {
	// Tạo OAuth2 config từ client id và client secret
	config := d.GetOAuthConfig()

	// Kiểm tra token đã lưu
	tokenFile := filepath.Join(d.Config.TokenDir, "token.json")
	token, err := d.tokenFromFile(tokenFile)

	// Nếu không có token hoặc token không hợp lệ
	if err != nil {
		return nil, fmt.Errorf("không tìm thấy token xác thực. Vui lòng xác thực qua UI hoặc CLI")
	}

	// Tạo service sử dụng token
	ctx := context.Background()
	service, err := drive.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
	if err != nil {
		return nil, fmt.Errorf("không thể tạo Drive service: %v", err)
	}

	return service, nil
}

// CheckAuth kiểm tra đã xác thực chưa
func (d *DriveUploader) CheckAuth() bool {
	tokenFile := filepath.Join(d.Config.TokenDir, "token.json")
	_, err := d.tokenFromFile(tokenFile)
	return err == nil
}

// tokenFromFile đọc token từ file
func (d *DriveUploader) tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	return token, err
}

// getTokenFromWeb yêu cầu người dùng xác thực qua trình duyệt
func (d *DriveUploader) getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	// Tạo URL xác thực
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\n=== HƯỚNG DẪN XÁC THỰC GOOGLE DRIVE ===\n")
	fmt.Printf("\nBước 1: Mở URL sau trong trình duyệt:\n")
	fmt.Printf("%s\n", authURL)
	fmt.Printf("\nBước 2: Đăng nhập Google và cho phép quyền truy cập\n")
	fmt.Printf("\nBước 3: Copy mã và paste vào đây\n\n")

	var code string
	fmt.Print("Nhập mã xác thực: ")
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("không thể đọc mã xác thực: %v", err)
	}

	// Đổi mã xác thực lấy token
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("không thể đổi mã xác thực: %v", err)
	}

	return token, nil
}

// saveToken lưu token vào file
func (d *DriveUploader) saveToken(path string, token *oauth2.Token) error {
	// Đảm bảo thư mục tồn tại
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	// Mở file để ghi
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Ghi token vào file dưới dạng JSON
	return json.NewEncoder(f).Encode(token)
}

// createOrFindFolder tạo hoặc tìm folder trên Drive
func (d *DriveUploader) createOrFindFolder(service *drive.Service, name string, parentID string) (string, error) {
	// Tạo query để tìm folder
	query := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder'", name)
	if parentID != "" {
		query += fmt.Sprintf(" and '%s' in parents", parentID)
	}

	// Tìm folder
	r, err := service.Files.List().Q(query).Fields("files(id, name)").Do()
	if err != nil {
		return "", fmt.Errorf("không thể tìm folder: %v", err)
	}

	// Nếu folder đã tồn tại
	if len(r.Files) > 0 {
		folderID := r.Files[0].Id
		fmt.Printf("Sử dụng folder có sẵn: %s (ID: %s)\n", name, folderID)
		return folderID, nil
	}

	// Nếu chưa có folder, tạo mới
	folderMetadata := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
	}

	// Nếu có parent folder, thiết lập parents
	if parentID != "" {
		folderMetadata.Parents = []string{parentID}
	}

	// Tạo folder
	folder, err := service.Files.Create(folderMetadata).Fields("id").Do()
	if err != nil {
		return "", fmt.Errorf("không thể tạo folder: %v", err)
	}

	fmt.Printf("Đã tạo folder mới: %s (ID: %s)\n", name, folder.Id)
	return folder.Id, nil
}

// checkFileExists kiểm tra file đã tồn tại trong folder chưa
func (d *DriveUploader) checkFileExists(service *drive.Service, fileName string, parentFolderID string) (bool, error) {
	query := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false", fileName, parentFolderID)
	r, err := service.Files.List().Q(query).Fields("files(id, name)").Do()
	if err != nil {
		return false, fmt.Errorf("không thể kiểm tra file: %v", err)
	}

	return len(r.Files) > 0, nil
}

// UploadFile upload một file lên Google Drive
func (d *DriveUploader) UploadFile(filePath string) error {
	// Lấy Drive client
	service, err := d.getClient()
	if err != nil {
		return fmt.Errorf("không thể kết nối Google Drive: %v", err)
	}

	// Tạo folder gốc nếu chưa có
	rootFolderID, err := d.createOrFindFolder(service, d.Config.FolderDrive, "")
	if err != nil {
		return fmt.Errorf("không thể tạo folder gốc: %v", err)
	}

	// Tạo folder theo ngày
	today := time.Now().Format("2006-01-02")
	dateFolderID, err := d.createOrFindFolder(service, today, rootFolderID)
	if err != nil {
		return fmt.Errorf("không thể tạo folder ngày: %v", err)
	}

	// Lấy tên file
	fileName := filepath.Base(filePath)

	// Kiểm tra file đã tồn tại chưa
	exists, err := d.checkFileExists(service, fileName, dateFolderID)
	if err != nil {
		return fmt.Errorf("không thể kiểm tra file tồn tại: %v", err)
	}

	if exists {
		fmt.Printf("File %s đã tồn tại trong thư mục, bỏ qua upload\n", fileName)
		return nil
	}

	// Chuẩn bị metadata
	fileMetadata := &drive.File{
		Name:    fileName,
		Parents: []string{dateFolderID},
	}

	// Mở file để upload
	content, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("không thể mở file: %v", err)
	}
	defer content.Close()

	// Upload file
	file, err := service.Files.Create(fileMetadata).
		Media(content).
		Fields("id, webViewLink").
		Do()
	if err != nil {
		return fmt.Errorf("không thể upload file: %v", err)
	}

	fmt.Printf("File %s đã được upload:\n", fileName)
	fmt.Printf("- File ID: %s\n", file.Id)
	fmt.Printf("- Web Link: %s\n", file.WebViewLink)

	return nil
}

// UploadAllBackups upload tất cả các file backup trong thư mục backups
func (d *DriveUploader) UploadAllBackups() error {
	// Lấy Drive client
	service, err := d.getClient()
	if err != nil {
		return fmt.Errorf("không thể kết nối Google Drive: %v", err)
	}

	// Tạo folder gốc nếu chưa có
	rootFolderID, err := d.createOrFindFolder(service, d.Config.FolderDrive, "")
	if err != nil {
		return fmt.Errorf("không thể tạo folder gốc: %v", err)
	}

	// Duyệt qua từng thư mục ngày
	backupDir := d.Config.BackupDir
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("không thể đọc thư mục backup: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dateFolderPath := filepath.Join(backupDir, entry.Name())

			// Tạo hoặc lấy folder ngày trên Drive
			dateFolderID, err := d.createOrFindFolder(service, entry.Name(), rootFolderID)
			if err != nil {
				fmt.Printf("Không thể tạo folder ngày %s: %v\n", entry.Name(), err)
				continue
			}

			// Đọc tất cả file SQL trong thư mục ngày
			files, err := filepath.Glob(filepath.Join(dateFolderPath, "*.sql"))
			if err != nil {
				fmt.Printf("Không thể đọc file trong thư mục %s: %v\n", dateFolderPath, err)
				continue
			}

			// Upload từng file
			for _, filePath := range files {
				fileName := filepath.Base(filePath)

				// Kiểm tra file đã tồn tại chưa
				exists, err := d.checkFileExists(service, fileName, dateFolderID)
				if err != nil {
					fmt.Printf("Không thể kiểm tra file %s: %v\n", fileName, err)
					continue
				}

				if exists {
					fmt.Printf("File %s đã tồn tại trên Drive, bỏ qua\n", fileName)
					continue
				}

				// Chuẩn bị metadata
				fileMetadata := &drive.File{
					Name:    fileName,
					Parents: []string{dateFolderID},
				}

				// Mở file để upload
				content, err := os.Open(filePath)
				if err != nil {
					fmt.Printf("Không thể mở file %s: %v\n", filePath, err)
					continue
				}

				// Upload file
				file, err := service.Files.Create(fileMetadata).
					Media(content).
					Fields("id").
					Do()
				content.Close()

				if err != nil {
					fmt.Printf("Không thể upload file %s: %v\n", fileName, err)
					continue
				}

				fmt.Printf("Đã upload file %s (ID: %s)\n", fileName, file.Id)
			}
		}
	}

	return nil
}
