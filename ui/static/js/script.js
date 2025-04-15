// JavaScript for the backup application
document.addEventListener('DOMContentLoaded', function() {
    // Tự động đóng thông báo alert sau 5 giây
    const alerts = document.querySelectorAll('.alert');
    if (alerts.length > 0) {
        setTimeout(function() {
            alerts.forEach(function(alert) {
                const closeButton = alert.querySelector('.btn-close');
                if (closeButton) {
                    closeButton.click();
                }
            });
        }, 5000);
    }

    // Thêm xác nhận trước khi upload tất cả
    const uploadAllForm = document.querySelector('form[action="/upload-all"]');
    if (uploadAllForm) {
        uploadAllForm.addEventListener('submit', function(e) {
            if (!confirm('Bạn có chắc chắn muốn upload tất cả các file backup lên Google Drive?')) {
                e.preventDefault();
            }
        });
    }

    // Format file size ở UI
    const fileSizeCells = document.querySelectorAll('td:nth-child(3)');
    fileSizeCells.forEach(function(cell) {
        const sizeInBytes = parseInt(cell.textContent);
        if (!isNaN(sizeInBytes)) {
            if (sizeInBytes < 1024) {
                cell.textContent = sizeInBytes + ' B';
            } else if (sizeInBytes < 1024 * 1024) {
                cell.textContent = (sizeInBytes / 1024).toFixed(2) + ' KB';
            } else if (sizeInBytes < 1024 * 1024 * 1024) {
                cell.textContent = (sizeInBytes / (1024 * 1024)).toFixed(2) + ' MB';
            } else {
                cell.textContent = (sizeInBytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
            }
        }
    });
    
    // Xử lý đóng cửa sổ callback OAuth tự động
    // Kiểm tra URL để biết chúng ta có ở trang callback không
    if (window.location.pathname === '/callback') {
        // Lấy query parameter 'success'
        const urlParams = new URLSearchParams(window.location.search);
        const hasSuccess = urlParams.has('success');
        
        if (hasSuccess) {
            // Thông báo cho cửa sổ chính làm mới
            if (window.opener && !window.opener.closed) {
                window.opener.postMessage('auth-success', '*');
            }
            
            // Đóng cửa sổ callback sau một khoảng thời gian ngắn
            setTimeout(function() {
                window.close();
            }, 500);
        }
    }
    
    // Lắng nghe tin nhắn từ cửa sổ callback
    window.addEventListener('message', function(event) {
        if (event.data === 'auth-success') {
            // Làm mới trang để cập nhật trạng thái xác thực
            window.location.reload();
        }
    });
}); 