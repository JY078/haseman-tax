package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"math/rand"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
)

type smtpConfig struct {
	host     string
	port     string
	username string
	password string
	fromAddr string
	fromName string
	useSSL   bool
}

func getSMTPConfig() (*smtpConfig, error) {
	keys := []string{
		"smtp_host", "smtp_port", "smtp_username",
		"smtp_password", "smtp_from_address", "smtp_from_name", "smtp_use_ssl",
	}

	var settings []model.Setting
	if err := database.DB.Where("\"key\" IN ?", keys).Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch SMTP settings: %w", err)
	}

	config := &smtpConfig{
		port:   "465",
		useSSL: true,
	}

	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}

	config.host = settingsMap["smtp_host"]
	if settingsMap["smtp_port"] != "" {
		config.port = settingsMap["smtp_port"]
	}
	config.username = settingsMap["smtp_username"]
	config.password = settingsMap["smtp_password"]
	config.fromAddr = settingsMap["smtp_from_address"]
	config.fromName = settingsMap["smtp_from_name"]
	if settingsMap["smtp_use_ssl"] != "" {
		config.useSSL = settingsMap["smtp_use_ssl"] == "true"
	}

	if config.host == "" || config.username == "" || config.password == "" {
		return nil, fmt.Errorf("SMTP configuration is incomplete")
	}

	return config, nil
}

func SendEmail(to, subject, body string) error {
	return sendEmail(to, subject, body, "", "", nil)
}

// SendEmailWithAttachment sends an HTML email with one attachment.
func SendEmailWithAttachment(to, subject, body, fileName, contentType string, attachment []byte) error {
	return sendEmail(to, subject, body, fileName, contentType, attachment)
}

func sendEmail(to, subject, body, fileName, contentType string, attachment []byte) error {
	config, err := getSMTPConfig()
	if err != nil {
		return err
	}

	// RFC 2047：非 ASCII 字符（如中文）必须编码，否则被大多数反垃圾系统判定为伪造头
	encodedSubject := mime.BEncoding.Encode("UTF-8", subject)

	// From 显示名含非 ASCII 字符时同样需要编码
	var fromHeader string
	if config.fromName != "" {
		encodedName := mime.BEncoding.Encode("UTF-8", config.fromName)
		fromHeader = fmt.Sprintf("%s <%s>", encodedName, config.fromAddr)
	} else {
		fromHeader = config.fromAddr
	}

	// Message-ID 使用时间戳+随机数确保全局唯一，缺失此头是强垃圾邮件信号
	msgID := fmt.Sprintf("<%d.%d@%s>", time.Now().UnixNano(), rand.Int63(), config.host)

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: %s\r\nMIME-Version: 1.0\r\n", fromHeader, to, encodedSubject, time.Now().Format(time.RFC1123Z), msgID)
	if fileName == "" {
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
		msg.WriteString(encodeBase64Body(body))
	} else {
		var multipartBody bytes.Buffer
		writer := multipart.NewWriter(&multipartBody)
		headers := make(textproto.MIMEHeader)
		headers.Set("Content-Type", "text/html; charset=UTF-8")
		headers.Set("Content-Transfer-Encoding", "base64")
		part, err := writer.CreatePart(headers)
		if err != nil {
			return err
		}
		if _, err = part.Write([]byte(encodeBase64Body(body))); err != nil {
			return err
		}
		attachmentHeaders := make(textproto.MIMEHeader)
		// The MCF filename is ASCII (MCF_LOWES_yyyyMMdd_HHmmss.txt). Keep the
		// attachment headers plain for maximum compatibility with QQ Mail and
		// other clients; RFC 2231 parameters are mis-rendered by some clients.
		if isASCIIFileName(fileName) {
			attachmentHeaders.Set("Content-Type", fmt.Sprintf("%s; name=\"%s\"", contentType, fileName))
			attachmentHeaders.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
		} else {
			fallbackName := asciiAttachmentFilename(fileName)
			attachmentHeaders.Set("Content-Type", fmt.Sprintf("%s; name=\"%s\"", contentType, fallbackName))
			attachmentHeaders.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fallbackName))
		}
		attachmentHeaders.Set("Content-Transfer-Encoding", "base64")
		attachmentPart, err := writer.CreatePart(attachmentHeaders)
		if err != nil {
			return err
		}
		if _, err = attachmentPart.Write([]byte(encodeBase64Body(string(attachment)))); err != nil {
			return err
		}
		if err = writer.Close(); err != nil {
			return err
		}
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n%s", writer.Boundary(), multipartBody.String()))
	}

	auth := smtp.PlainAuth("", config.username, config.password, config.host)
	addr := fmt.Sprintf("%s:%s", config.host, config.port)

	if config.useSSL {
		tlsConfig := &tls.Config{
			ServerName: config.host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, config.host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Quit()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
		if err = client.Mail(config.fromAddr); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}
		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to create data writer: %w", err)
		}
		defer w.Close()

		if _, err = w.Write(msg.Bytes()); err != nil {
			return fmt.Errorf("failed to write email: %w", err)
		}
	} else {
		if err := smtp.SendMail(addr, auth, config.fromAddr, []string{to}, msg.Bytes()); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	return nil
}

func asciiAttachmentFilename(fileName string) string {
	var b strings.Builder
	for _, r := range fileName {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._-", r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "attachment"
	}
	return b.String()
}

func isASCIIFileName(fileName string) bool {
	for _, r := range fileName {
		if r > 0x7f || strings.ContainsRune("\r\n\"", r) {
			return false
		}
	}
	return fileName != ""
}

func SendVerificationCode(to, code string) error {
	subject := "邮箱验证码"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
<style>
  .container { max-width: 600px; margin: 0 auto; padding: 20px; font-family: Arial, sans-serif; }
  .code { font-size: 32px; font-weight: bold; color: #1a1a1a; letter-spacing: 4px; padding: 15px 30px; background: #f5f5f5; display: inline-block; border-radius: 8px; }
  .note { color: #666; font-size: 14px; margin-top: 20px; }
</style>
</head>
<body>
  <div class="container">
    <h2>邮箱验证码</h2>
    <p>您的验证码是：</p>
    <div class="code">%s</div>
    <p class="note">验证码有效期为 5 分钟，请勿将验证码告知他人。</p>
    <p class="note">如非本人操作，请忽略此邮件。</p>
  </div>
</body>
</html>`, code)

	return SendEmail(to, subject, body)
}

func SendPasswordResetCode(to, code string) error {
	subject := "密码重置验证码"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
<style>
  .container { max-width: 600px; margin: 0 auto; padding: 20px; font-family: Arial, sans-serif; }
  .code { font-size: 32px; font-weight: bold; color: #1a1a1a; letter-spacing: 4px; padding: 15px 30px; background: #f5f5f5; display: inline-block; border-radius: 8px; }
  .note { color: #666; font-size: 14px; margin-top: 20px; }
  .warn { color: #e53e3e; font-size: 13px; margin-top: 10px; }
</style>
</head>
<body>
  <div class="container">
    <h2>密码重置验证码</h2>
    <p>我们收到了您的密码重置请求，验证码是：</p>
    <div class="code">%s</div>
    <p class="note">验证码有效期为 10 分钟，请尽快完成密码重置。</p>
    <p class="warn">如非本人操作，请立即登录修改密码，确保账号安全。</p>
  </div>
</body>
</html>`, code)

	return SendEmail(to, subject, body)
}

func GenerateVerificationCode() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := make([]byte, 6)
	for i := range code {
		code[i] = byte('0' + r.Intn(10))
	}
	return string(code)
}

func IsEmailExists(email string) bool {
	var count int64
	database.DB.Table("users").Where("LOWER(email) = ?", strings.ToLower(email)).Count(&count)
	return count > 0
}

// encodeBase64Body 将邮件正文编码为 Base64，并按 RFC 2045 要求每 76 字符换行
func encodeBase64Body(body string) string {
	const lineLen = 76
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	var buf strings.Builder
	for i := 0; i < len(encoded); i += lineLen {
		end := i + lineLen
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.WriteString(encoded[i:end])
		buf.WriteString("\r\n")
	}
	return buf.String()
}
