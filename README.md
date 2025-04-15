# Backup Database và Upload Google Drive

Ứng dụng này cung cấp các tính năng:
1. Dump database từ container Docker
2. Upload các file dump lên Google Drive
3. Giao diện web quản lý và theo dõi

## Yêu cầu

- Go 1.21 trở lên
- Docker (để truy cập container database)
- Tài khoản Google Drive

## Cấu hình

Tạo file `.env` trong thư mục gốc với nội dung:

```
DB_USER=postgres
DB_PASSWORD=postgres
CONTAINER_NAME=tên-container-postgres
DB_NAME=tên-database
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
FOLDER_DRIVE=tên-folder-trên-drive
CRON_SCHEDULE=*/5 * * * *
```

## Cài đặt

```bash
# Clone repository
git clone https://github.com/yourusername/go-backup.git
cd go-backup

# Cài đặt dependencies
go mod download
```

## Sử dụng

### Dòng lệnh

```bash
# Chỉ dump database
go run cmd/backup/main.go --dump

# Upload file backup mới nhất
go run cmd/backup/main.go --upload-last

# Upload tất cả file backup
go run cmd/backup/main.go --upload-all
```

### Chạy ứng dụng web

```bash
go run cmd/backup/main.go --web --port 8080
```

Sau đó truy cập `http://localhost:8080` để sử dụng giao diện web.

## Xác thực Google Drive

Lần đầu tiên sử dụng tính năng upload, ứng dụng sẽ yêu cầu xác thực với Google Drive:

1. Mở URL được hiển thị trong terminal
2. Đăng nhập và cấp quyền truy cập
3. Sao chép mã xác thực và dán vào terminal

## Cấu trúc thư mục

```
go-backup/
├── cmd/
│   └── backup/
│       └── main.go          # File chính để chạy ứng dụng
├── internal/
│   ├── config/              # Xử lý cấu hình
│   ├── dbdump/              # Xử lý dump database
│   ├── drive/               # Xử lý upload lên Drive
│   ├── handlers/            # Xử lý HTTP request
│   └── models/              # Cấu trúc dữ liệu
├── ui/
│   ├── static/              # CSS, JavaScript
│   └── templates/           # HTML templates
├── token/                   # Lưu token Google Drive
├── backups/                 # Thư mục lưu file backup
├── .env                     # Cấu hình
└── go.mod                   # Go modules
```

## Đóng góp

Vui lòng tạo issue hoặc pull request để đóng góp cho dự án. 