package util

import (
	"log"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Warning string `json:"warning,omitempty"`
}

// HandleError logs the detailed error and returns a generic error message to the user
func HandleError(c *gin.Context, statusCode int, userMessage string, detailedError error) {
	// Log the detailed error for debugging
	if detailedError != nil {
		log.Printf("[ERROR] %s: %v", userMessage, detailedError)
	}

	// Return generic error to user
	c.JSON(statusCode, ErrorResponse{
		Error: userMessage,
	})
}

// HandleErrorWithWarning logs the detailed error and returns a generic error message with warning
func HandleErrorWithWarning(c *gin.Context, statusCode int, userMessage string, warning string, detailedError error) {
	// Log the detailed error for debugging
	if detailedError != nil {
		log.Printf("[ERROR] %s: %v", userMessage, detailedError)
	}

	// Return generic error with warning to user
	c.JSON(statusCode, ErrorResponse{
		Error:   userMessage,
		Warning: warning,
	})
}

// Common error messages for different scenarios
const (
	ErrFileUploadFailed    = "Không thể tải lên file"
	ErrFileTooLarge        = "File quá lớn, chỉ hỗ trợ file tối đa 100MB"
	ErrFileDurationTooLong = "Chỉ cho phép video/audio dưới 7 phút"
	ErrInvalidFileType     = "Loại file không được hỗ trợ"
	ErrDirectoryCreation   = "Không thể tạo thư mục làm việc"
	ErrDatabaseOperation   = "Lỗi hệ thống, vui lòng thử lại"
	ErrServiceUnavailable  = "Dịch vụ tạm thời không khả dụng"
	ErrInsufficientCredits = "Không đủ credit để sử dụng dịch vụ"
	ErrProcessingFailed    = "Xử lý thất bại, vui lòng thử lại"
	ErrUnauthorized        = "Không có quyền truy cập"
	ErrInvalidRequest      = "Yêu cầu không hợp lệ"
	ErrInternalServer      = "Lỗi hệ thống, vui lòng thử lại sau"
)

// Common warning messages
const (
	WarningInsufficientCredits = "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!"
)
