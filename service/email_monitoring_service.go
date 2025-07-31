package service

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"creator-tool-backend/config"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// EmailMonitoringService quản lý việc đọc mail xác nhận thanh toán
type EmailMonitoringService struct {
	client      *client.Client
	emailConfig EmailConfig
}

// EmailConfig cấu hình email
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	SSL      bool
}

// PaymentConfirmation thông tin xác nhận thanh toán từ mail
type PaymentConfirmation struct {
	OrderCode     string
	Amount        string
	BankAccount   string
	TransactionID string
	EmailContent  string
	ReceivedAt    time.Time
}

// NewEmailMonitoringService tạo instance mới
func NewEmailMonitoringService(config EmailConfig) *EmailMonitoringService {
	return &EmailMonitoringService{
		emailConfig: config,
	}
}

// ConnectIMAP kết nối đến mail server
func (s *EmailMonitoringService) ConnectIMAP() error {
	var err error
	if s.emailConfig.SSL {
		s.client, err = client.DialTLS(fmt.Sprintf("%s:%d", s.emailConfig.Host, s.emailConfig.Port), nil)
	} else {
		s.client, err = client.Dial(fmt.Sprintf("%s:%d", s.emailConfig.Host, s.emailConfig.Port))
	}
	if err != nil {
		return fmt.Errorf("failed to connect to mail server: %v", err)
	}

	// Login
	err = s.client.Login(s.emailConfig.Username, s.emailConfig.Password)
	if err != nil {
		return fmt.Errorf("failed to login: %v", err)
	}

	log.Printf("Connected to mail server: %s", s.emailConfig.Host)
	return nil
}

// ReadNewEmails đọc mail mới
func (s *EmailMonitoringService) ReadNewEmails() error {
	if s.client == nil {
		return fmt.Errorf("not connected to mail server")
	}

	// Chọn mailbox INBOX
	_, err := s.client.Select("INBOX", false)
	if err != nil {
		return fmt.Errorf("failed to select INBOX: %v", err)
	}

	// Lấy mail mới (từ 1 giờ trước)
	since := time.Now().Add(-1 * time.Hour)
	criteria := imap.NewSearchCriteria()
	criteria.Since = since

	uids, err := s.client.Search(criteria)
	if err != nil {
		return fmt.Errorf("failed to search emails: %v", err)
	}

	if len(uids) == 0 {
		return nil // Không có mail mới
	}

	// Lấy nội dung mail
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- s.client.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBody, imap.FetchUid}, messages)
	}()

	for msg := range messages {
		// Chỉ xử lý email từ ACB
		sender := msg.Envelope.From[0].Address()
		if !strings.Contains(strings.ToLower(sender), "acb") {
			continue
		}

		confirmation, err := s.parsePaymentEmail(msg)
		if err != nil {
			log.Printf("Failed to parse email: %v", err)
			// Lưu log lỗi parse
			savePaymentEmailLog(config.PaymentEmailLog{
				Sender:       sender,
				Subject:      msg.Envelope.Subject,
				EmailContent: s.getEmailBody(msg),
				Status:       "error",
				ErrorMessage: err.Error(),
				ReceivedAt:   msg.Envelope.Date,
				CreatedAt:    time.Now(),
			})
			continue
		}

		if confirmation != nil {
			err = s.processPaymentEmail(confirmation, sender, msg.Envelope.Subject, s.getEmailBody(msg), msg.Envelope.Date)
			if err != nil {
				log.Printf("Failed to process payment email: %v", err)
			}
		}
	}

	return <-done
}

// parsePaymentEmail parse mail xác nhận thanh toán
func (s *EmailMonitoringService) parsePaymentEmail(msg *imap.Message) (*PaymentConfirmation, error) {
	// Kiểm tra sender
	sender := msg.Envelope.From[0].Address()
	if !s.isBankEmail(sender) {
		return nil, nil // Không phải mail ngân hàng
	}

	// Lấy nội dung mail
	body := s.getEmailBody(msg)
	if body == "" {
		return nil, fmt.Errorf("empty email body")
	}

	// Parse thông tin thanh toán
	confirmation, err := s.extractPaymentInfo(body, sender)
	if err != nil {
		return nil, err
	}

	return confirmation, nil
}

// isBankEmail kiểm tra có phải mail ngân hàng không
func (s *EmailMonitoringService) isBankEmail(sender string) bool {
	bankEmails := []string{
		"noreply@vietcombank.com.vn",
		"noreply@bidv.com.vn",
		"noreply@tcb.com.vn",
		"noreply@acb.com.vn",
		"noreply@mb.com.vn",
		"mailalert@acb.com.vn",
	}

	for _, bankEmail := range bankEmails {
		if strings.Contains(strings.ToLower(sender), strings.ToLower(bankEmail)) {
			return true
		}
	}

	return false
}

// getEmailBody lấy nội dung mail
func (s *EmailMonitoringService) getEmailBody(msg *imap.Message) string {
	// Đơn giản hóa: trả về placeholder cho bây giờ
	// Trong thực tế cần implement đọc MIME content đầy đủ
	// TODO: Implement proper email body reading
	return "Email content placeholder - cần implement đọc MIME content"
}

// extractPaymentInfo trích xuất thông tin thanh toán từ nội dung mail
func (s *EmailMonitoringService) extractPaymentInfo(content, sender string) (*PaymentConfirmation, error) {
	// Template cho Vietcombank
	if strings.Contains(sender, "vietcombank") {
		return s.parseVietcombankEmail(content)
	}

	// Template cho BIDV
	if strings.Contains(sender, "bidv") {
		return s.parseBIDVEmail(content)
	}

	// Template cho ACB
	if strings.Contains(sender, "acb") {
		return s.parseACBEmail(content)
	}

	// Template mặc định
	return s.parseDefaultEmail(content)
}

// parseVietcombankEmail parse mail Vietcombank
func (s *EmailMonitoringService) parseVietcombankEmail(content string) (*PaymentConfirmation, error) {
	// Regex patterns cho Vietcombank
	amountPattern := regexp.MustCompile(`Số tiền:\s*([\d,]+)\s*VND`)
	accountPattern := regexp.MustCompile(`Tài khoản:\s*(\d+)`)
	orderPattern := regexp.MustCompile(`Nội dung:\s*([A-Z0-9]+)`)

	amountMatch := amountPattern.FindStringSubmatch(content)
	accountMatch := accountPattern.FindStringSubmatch(content)
	orderMatch := orderPattern.FindStringSubmatch(content)

	if len(amountMatch) < 2 || len(accountMatch) < 2 || len(orderMatch) < 2 {
		return nil, fmt.Errorf("failed to extract payment info from Vietcombank email")
	}

	return &PaymentConfirmation{
		OrderCode:     orderMatch[1],
		Amount:        strings.ReplaceAll(amountMatch[1], ",", ""),
		BankAccount:   accountMatch[1],
		TransactionID: fmt.Sprintf("VCB_%d", time.Now().Unix()),
		EmailContent:  content,
		ReceivedAt:    time.Now(),
	}, nil
}

// parseBIDVEmail parse mail BIDV
func (s *EmailMonitoringService) parseBIDVEmail(content string) (*PaymentConfirmation, error) {
	// Regex patterns cho BIDV
	amountPattern := regexp.MustCompile(`Số tiền:\s*([\d,]+)\s*VNĐ`)
	accountPattern := regexp.MustCompile(`Tài khoản:\s*(\d+)`)
	orderPattern := regexp.MustCompile(`Nội dung:\s*([A-Z0-9]+)`)

	amountMatch := amountPattern.FindStringSubmatch(content)
	accountMatch := accountPattern.FindStringSubmatch(content)
	orderMatch := orderPattern.FindStringSubmatch(content)

	if len(amountMatch) < 2 || len(accountMatch) < 2 || len(orderMatch) < 2 {
		return nil, fmt.Errorf("failed to extract payment info from BIDV email")
	}

	return &PaymentConfirmation{
		OrderCode:     orderMatch[1],
		Amount:        strings.ReplaceAll(amountMatch[1], ",", ""),
		BankAccount:   accountMatch[1],
		TransactionID: fmt.Sprintf("BIDV_%d", time.Now().Unix()),
		EmailContent:  content,
		ReceivedAt:    time.Now(),
	}, nil
}

// parseACBEmail parse mail ACB
func (s *EmailMonitoringService) parseACBEmail(content string) (*PaymentConfirmation, error) {
	// Regex patterns cho ACB dựa trên email mẫu
	amountPattern := regexp.MustCompile(`Ghi có\s*\+([\d,]+\.?\d*)\s*VND`)
	accountPattern := regexp.MustCompile(`tài khoản\s*(\d+)\s*của`)
	orderPattern := regexp.MustCompile(`PAY\d{14,20}`) // Cập nhật để linh hoạt hơn
	transactionPattern := regexp.MustCompile(`FT\d+`)

	amountMatch := amountPattern.FindStringSubmatch(content)
	accountMatch := accountPattern.FindStringSubmatch(content)
	orderMatch := orderPattern.FindString(content)
	transactionMatch := transactionPattern.FindString(content)

	if orderMatch == "" {
		return nil, fmt.Errorf("no order code found in ACB email")
	}

	if len(amountMatch) < 2 {
		return nil, fmt.Errorf("failed to extract amount from ACB email")
	}

	bankAccount := "Unknown"
	if len(accountMatch) >= 2 {
		bankAccount = accountMatch[1]
	}

	transactionID := fmt.Sprintf("ACB_%d", time.Now().Unix())
	if transactionMatch != "" {
		transactionID = transactionMatch
	}

	return &PaymentConfirmation{
		OrderCode:     orderMatch,
		Amount:        strings.ReplaceAll(amountMatch[1], ",", ""),
		BankAccount:   bankAccount,
		TransactionID: transactionID,
		EmailContent:  content,
		ReceivedAt:    time.Now(),
	}, nil
}

// parseDefaultEmail parse mail mặc định
func (s *EmailMonitoringService) parseDefaultEmail(content string) (*PaymentConfirmation, error) {
	// Tìm order code trong nội dung
	orderPattern := regexp.MustCompile(`PAY\d{14,20}`)
	orderMatch := orderPattern.FindString(content)

	if orderMatch == "" {
		return nil, fmt.Errorf("no order code found in email")
	}

	// Tìm số tiền
	amountPattern := regexp.MustCompile(`(\d{1,3}(?:,\d{3})*)`)
	amountMatches := amountPattern.FindAllString(content, -1)

	var amount string
	for _, match := range amountMatches {
		// Tìm số tiền lớn nhất (có thể là số tiền thanh toán)
		if len(match) > len(amount) {
			amount = match
		}
	}

	return &PaymentConfirmation{
		OrderCode:     orderMatch,
		Amount:        strings.ReplaceAll(amount, ",", ""),
		BankAccount:   "Unknown",
		TransactionID: fmt.Sprintf("BANK_%d", time.Now().Unix()),
		EmailContent:  content,
		ReceivedAt:    time.Now(),
	}, nil
}

// processPaymentEmail xử lý mail xác nhận thanh toán
func (s *EmailMonitoringService) processPaymentEmail(confirmation *PaymentConfirmation, sender, subject, emailContent string, receivedAt time.Time) error {
	logEntry := config.PaymentEmailLog{
		OrderCode:     confirmation.OrderCode,
		Amount:        confirmation.Amount,
		BankAccount:   confirmation.BankAccount,
		TransactionID: confirmation.TransactionID,
		Sender:        sender,
		Subject:       subject,
		EmailContent:  emailContent,
		ReceivedAt:    receivedAt,
		CreatedAt:     time.Now(),
	}

	// Tìm đơn hàng theo order code
	paymentService := NewPaymentOrderService()
	order, err := paymentService.GetOrderByCode(confirmation.OrderCode)
	if err != nil {
		logEntry.Status = "unmatched"
		logEntry.ErrorMessage = "order not found: " + confirmation.OrderCode
		savePaymentEmailLog(logEntry)
		return fmt.Errorf(logEntry.ErrorMessage)
	}
	logEntry.OrderID = &order.ID

	// Kiểm tra trạng thái đơn hàng
	if order.OrderStatus != "pending" {
		logEntry.Status = "error"
		logEntry.ErrorMessage = fmt.Sprintf("order %s is not pending", confirmation.OrderCode)
		savePaymentEmailLog(logEntry)
		return fmt.Errorf(logEntry.ErrorMessage)
	}

	// Kiểm tra số tiền
	expectedAmount := order.AmountVND.String()
	if confirmation.Amount != expectedAmount {
		logEntry.Status = "error"
		logEntry.ErrorMessage = fmt.Sprintf("amount mismatch: expected %s, got %s", expectedAmount, confirmation.Amount)
		savePaymentEmailLog(logEntry)
		return fmt.Errorf(logEntry.ErrorMessage)
	}

	// Cập nhật trạng thái đơn hàng thành paid
	err = paymentService.UpdateOrderStatus(order.ID, "paid", &confirmation.TransactionID)
	if err != nil {
		logEntry.Status = "error"
		logEntry.ErrorMessage = fmt.Sprintf("failed to update order status: %v", err)
		savePaymentEmailLog(logEntry)
		return fmt.Errorf(logEntry.ErrorMessage)
	}

	// Cộng credit cho user
	creditService := NewCreditService()
	amountUSD, _ := order.AmountUSD.Float64()
	err = creditService.AddCredits(order.UserID, amountUSD,
		fmt.Sprintf("Nạp credit qua thanh toán QR - %s", confirmation.OrderCode),
		confirmation.TransactionID)
	if err != nil {
		logEntry.Status = "error"
		logEntry.ErrorMessage = fmt.Sprintf("failed to add credits: %v", err)
		savePaymentEmailLog(logEntry)
		return fmt.Errorf(logEntry.ErrorMessage)
	}

	logEntry.Status = "matched"
	logEntry.ErrorMessage = ""
	savePaymentEmailLog(logEntry)

	log.Printf("Payment confirmed for order %s: %s USD", confirmation.OrderCode, order.AmountUSD.String())
	return nil
}

// Lưu log email thanh toán vào DB
func savePaymentEmailLog(log config.PaymentEmailLog) {
	_ = config.Db.Create(&log).Error
}

// StartEmailWorker khởi động worker đọc mail
func (s *EmailMonitoringService) StartEmailWorker() error {
	// Kết nối mail server
	err := s.ConnectIMAP()
	if err != nil {
		return fmt.Errorf("failed to connect to mail server: %v", err)
	}

	// Chạy worker trong goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Đọc mail mỗi 30 giây
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.ReadNewEmails(); err != nil {
					log.Printf("Error reading emails: %v", err)
				}
			}
		}
	}()

	log.Println("Email monitoring worker started")
	return nil
}

// Close đóng kết nối mail
func (s *EmailMonitoringService) Close() error {
	if s.client != nil {
		return s.client.Logout()
	}
	return nil
}
