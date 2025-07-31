package service

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/skip2/go-qrcode"
)

type QRService struct{}

func NewQRService() *QRService {
	return &QRService{}
}

// GenerateVietQRCode tạo QR code cho thanh toán VietQR theo chuẩn NAPAS247
func (s *QRService) GenerateVietQRCode(bankCode, accountNumber, amount, orderCode string) (string, error) {
	// Tạo VietQR payload theo chuẩn NAPAS247
	params := ParamsQrCode{
		SERVICE:      SERVICE_VA_ORDER, // QR-ĐỘNG
		BANK_ACCOUNT: accountNumber,
		CARDBIN:      bankCode, // VD: 970436 cho Vietcombank
		AMOUNT:       amount,
		CONTENT:      orderCode, // Nội dung thanh toán
	}

	// Generate VietQR string
	qrString := GenerateVietQR247(params)

	// Tạo QR code từ chuỗi VietQR
	qrCode, err := qrcode.Encode(qrString, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %v", err)
	}

	// Convert QR code thành base64
	base64String := base64.StdEncoding.EncodeToString(qrCode)
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64String)

	return dataURL, nil
}

// GeneratePaymentQRCode tạo QR code cho thông tin thanh toán
func (s *QRService) GeneratePaymentQRCode(bankName, accountNumber, accountName, amount, orderCode string) (string, error) {
	// Tạo chuỗi thông tin thanh toán
	qrString := fmt.Sprintf("Ngân hàng: %s\nTài khoản: %s\nChủ tài khoản: %s\nSố tiền: %s VND\nMã đơn hàng: %s\nThời gian: %s",
		bankName,
		accountNumber,
		accountName,
		amount,
		orderCode,
		time.Now().Format("02/01/2006 15:04:05"),
	)

	// Tạo QR code từ chuỗi
	qrCode, err := qrcode.Encode(qrString, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %v", err)
	}

	// Convert QR code thành base64
	base64String := base64.StdEncoding.EncodeToString(qrCode)
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64String)

	return dataURL, nil
}

// GenerateSimpleQRCode tạo QR code đơn giản từ text
func (s *QRService) GenerateSimpleQRCode(text string) (string, error) {
	// Tạo QR code từ text
	qrCode, err := qrcode.Encode(text, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %v", err)
	}

	// Convert QR code thành base64
	base64String := base64.StdEncoding.EncodeToString(qrCode)
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64String)

	return dataURL, nil
}
