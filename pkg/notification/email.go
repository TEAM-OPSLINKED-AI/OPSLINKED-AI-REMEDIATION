package notification

import (
	"crypto/tls" // crypto/tls가 필요합니다.
	"fmt"
	"net/smtp"
	"strings"

	"opslinked-ai/remediation-module/pkg/config"
	"opslinked-ai/remediation-module/pkg/types"
)

// Changed: SMTPS (Implicit SSL/TLS on port 465) 방식으로 전체 로직 변경
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

	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", cfg.FromEmail, strings.Join(to, ","), subject, body))

	// 2. 인증 메커니즘 생성
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	// 3. TLS 설정 구성
	tlsConfig := &tls.Config{
		ServerName: cfg.Host,
	}

	// 4. tls.Dial을 사용하여 SSL로 직접 연결 (STARTTLS 아님)
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to dial TLS (SMTPS): %w", err)
	}

	// 5. 보안 연결로부터 SMTP 클라이언트 생성
	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client from TLS connection: %w", err)
	}
	defer client.Close()

	// 6. 이미 보안 상태이므로 STARTTLS 없이 바로 인증
	if err = client.Auth(auth); err != nil {
		// 여기서도 500 에러가 나면, Naver가 AUTH PLAIN을 지원하지 않는 것입니다.
		// 하지만 535(비밀번호 오류)가 나면 성공입니다.
		return fmt.Errorf("failed to authenticate (over SMTPS): %w", err)
	}

	// 7. 메일 발송 (MAIL FROM, RCPT TO, DATA)
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

	// 8. 연결 종료
	client.Quit()
	return nil
}