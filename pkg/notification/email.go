package notification

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"opslinked-ai/remediation-module/pkg/config"
	"opslinked-ai/remediation-module/pkg/types"
)

func SendEmailNotification(cfg config.SMTPConfig, action types.RemediationAction, success bool, details string) error {
	if cfg.Host == "" || cfg.ToEmail == "" || cfg.FromEmail == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	to := strings.Split(cfg.ToEmail, ",")
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var status string
	if success {
		status = "SUCCESS"
	} else {
		status = "FAILURE"
	}

	// 1. 이메일 헤더 및 본문 구성
	subject := fmt.Sprintf("AIOps Remediation Action Report: %s - %s", status, action.ActionType)
	body := fmt.Sprintf(`
AIOps Remediation Action has been executed.

Status: %s
Action Type: %s
Namespace: %s
Resource: %s
Reason: %s
Triggered By: %s

Details:
%s
`, status, action.ActionType, action.Namespace, action.ResourceName, action.Reason, action.TriggeredBy, details)

	// RFC 822 형식의 이메일 메시지 생성
	// 'From'과 'To' 헤더가 여기에 포함되어야 합니다.
	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", cfg.FromEmail, strings.Join(to, ","), subject, body))

	// 2. smtp.Dial을 사용하여 서버에 연결
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to dial SMTP server: %w", err)
	}
	defer client.Close()

	// 3. STARTTLS로 보안 연결 시작
	// TLS 설정: 네이버는 유효한 인증서를 사용하므로 InsecureSkipVerify: false (기본값)
	tlsConfig := &tls.Config{
		ServerName: cfg.Host,
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// 4. PLAIN 인증 수행
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// 5. 메일 발송 (MAIL FROM, RCPT TO, DATA)
	if err = client.Mail(cfg.FromEmail); err != nil {
		return fmt.Errorf("failed to set MAIL FROM: %w", err)
	}
	for _, toAddr := range to {
		if err = client.Rcpt(toAddr); err != nil {
			return fmt.Errorf("failed to set RCPT TO for %s: %w", toAddr, err)
		}
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get DATA writer: %w", err)
	}
	
	_, err = wc.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}
	
	err = wc.Close()
	if err != nil {
		return fmt.Errorf("failed to close DATA writer: %w", err)
	}

	// 6. 연결 종료
	client.Quit()
	return nil
}