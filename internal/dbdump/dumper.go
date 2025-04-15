package dbdump

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/backup-cronjob/internal/config"
)

// DumpResult chứa thông tin kết quả dump
type DumpResult struct {
	FilePath string
	FileSize int64
	Success  bool
	Message  string
}

// DatabaseDumper là struct quản lý việc dump database
type DatabaseDumper struct {
	Config *config.Config
}

// NewDatabaseDumper tạo instance mới của DatabaseDumper
func NewDatabaseDumper(cfg *config.Config) *DatabaseDumper {
	return &DatabaseDumper{
		Config: cfg,
	}
}

// DumpDatabase thực hiện việc dump database từ container Docker
func (d *DatabaseDumper) DumpDatabase() (*DumpResult, error) {
	result := &DumpResult{
		Success: false,
	}

	// Tạo thư mục backup theo ngày
	now := time.Now()
	dateFolder := now.Format("2006-01-02")
	timestamp := now.Format("20060102_150405")

	backupDir := filepath.Join(d.Config.BackupDir, dateFolder)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		errMsg := fmt.Sprintf("Không thể tạo thư mục backup: %v", err)
		result.Message = errMsg
		return result, fmt.Errorf(errMsg)
	}

	fmt.Printf("Đã tạo thư mục backup: %s\n", backupDir)

	// Tạo tên file output
	outputFile := filepath.Join(backupDir, fmt.Sprintf("%s_%s_data.sql", d.Config.DBName, timestamp))

	// Tạo lệnh dump database
	cmd := exec.Command(
		"docker", "exec",
		"-e", fmt.Sprintf("PGPASSWORD=%s", d.Config.DBPassword),
		d.Config.ContainerName,
		"pg_dump",
		"-v",
		"--data-only",
		"--column-inserts",
		"--disable-triggers",
		"-U", d.Config.DBUser,
		"-d", d.Config.DBName,
	)

	fmt.Println("Đang thực hiện lệnh dump...")

	// Tạo file output
	outFile, err := os.Create(outputFile)
	if err != nil {
		errMsg := fmt.Sprintf("Không thể tạo file output: %v", err)
		result.Message = errMsg
		return result, fmt.Errorf(errMsg)
	}
	defer outFile.Close()

	// Thiết lập output, stderr
	cmd.Stdout = outFile
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		errMsg := fmt.Sprintf("Không thể thiết lập stderr pipe: %v", err)
		result.Message = errMsg
		return result, fmt.Errorf(errMsg)
	}

	// Thực thi lệnh
	if err := cmd.Start(); err != nil {
		errMsg := fmt.Sprintf("Không thể khởi động lệnh: %v", err)
		result.Message = errMsg
		return result, fmt.Errorf(errMsg)
	}

	// Đọc stderr
	stderrBytes, _ := io.ReadAll(stderrPipe)
	stderrOutput := string(stderrBytes)

	// Đợi lệnh hoàn thành
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Lỗi trong stderr: %s\n", stderrOutput)
		errMsg := fmt.Sprintf("Lệnh thất bại với mã lỗi: %v", err)
		result.Message = errMsg
		return result, fmt.Errorf(errMsg)
	}

	if stderrOutput != "" {
		fmt.Printf("Thông báo từ stderr: %s\n", stderrOutput)
	}

	// Kiểm tra file có tồn tại không
	fileInfo, err := os.Stat(outputFile)
	if err != nil {
		errMsg := fmt.Sprintf("File không được tạo tại %s: %v", outputFile, err)
		result.Message = errMsg
		return result, fmt.Errorf(errMsg)
	}

	fileSize := fileInfo.Size()

	fmt.Println("Dump dữ liệu thành công.")
	fmt.Printf("Vị trí file: %s\n", outputFile)
	fmt.Printf("Kích thước file: %.2f MB\n", float64(fileSize)/(1024*1024))

	result.FilePath = outputFile
	result.FileSize = fileSize
	result.Success = true
	result.Message = "Dump dữ liệu thành công"

	return result, nil
}
