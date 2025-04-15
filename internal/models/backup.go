package models

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupFile đại diện cho một file backup
type BackupFile struct {
	ID        string
	Name      string
	Path      string
	Size      int64
	CreatedAt time.Time
	Uploaded  bool
}

// FormatSize trả về kích thước file đã được format
func (b *BackupFile) FormatSize() string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	size := float64(b.Size)

	switch {
	case b.Size >= GB:
		return fmt.Sprintf("%.2f GB", size/GB)
	case b.Size >= MB:
		return fmt.Sprintf("%.2f MB", size/MB)
	case b.Size >= KB:
		return fmt.Sprintf("%.2f KB", size/KB)
	default:
		return fmt.Sprintf("%d B", b.Size)
	}
}

// FormatCreatedAt định dạng thời gian tạo
func (b *BackupFile) FormatCreatedAt() string {
	return b.CreatedAt.Format("02/01/2006 15:04:05")
}

// GetAllBackups lấy tất cả các file backup từ thư mục
func GetAllBackups(backupDir string) ([]*BackupFile, error) {
	var backups []*BackupFile

	// Duyệt qua tất cả thư mục con (thư mục ngày)
	dateDirs, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc thư mục backup: %v", err)
	}

	for _, dateDir := range dateDirs {
		if dateDir.IsDir() {
			dateDirPath := filepath.Join(backupDir, dateDir.Name())

			// Đọc tất cả file SQL trong thư mục ngày
			files, err := filepath.Glob(filepath.Join(dateDirPath, "*.sql"))
			if err != nil {
				continue
			}

			// Thêm mỗi file vào danh sách
			for _, file := range files {
				fileInfo, err := os.Stat(file)
				if err != nil {
					continue
				}

				backup := &BackupFile{
					ID:        filepath.Base(file),
					Name:      filepath.Base(file),
					Path:      file,
					Size:      fileInfo.Size(),
					CreatedAt: fileInfo.ModTime(),
					Uploaded:  false, // Sẽ được cập nhật sau
				}

				backups = append(backups, backup)
			}
		}
	}

	return backups, nil
}

// FindLatestBackup tìm file backup mới nhất
func FindLatestBackup(backupDir string) (*BackupFile, error) {
	backups, err := GetAllBackups(backupDir)
	if err != nil {
		return nil, err
	}

	if len(backups) == 0 {
		return nil, fmt.Errorf("không tìm thấy file backup nào")
	}

	// Tìm file mới nhất
	latest := backups[0]
	for _, backup := range backups {
		if backup.CreatedAt.After(latest.CreatedAt) {
			latest = backup
		}
	}

	return latest, nil
}
