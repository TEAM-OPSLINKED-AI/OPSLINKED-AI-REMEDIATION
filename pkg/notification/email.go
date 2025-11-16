package notification

import (
    "crypto/tls"
    "fmt"
    "net/smtp"
    "strings"

    "opslinked-ai/remediation-module/pkg/config"
    "opslinked-ai/remediation-module/pkg/types"
)

// SendEmailNotification은 STARTTLS(587) + AUTH PLAIN으로 SMTP 서버에 메일을 보냅니다.
func SendEmailNotification(cfg config.SMTPConfig, action types.RemediationAction, success bool, details string) error {
    if cfg.Host == "" || cfg.ToEmail == "" || cfg.FromEmail == "" {
        return fmt.Errorf("SMTP configuration is incomplete")
    }

    // 수신자 목록
    to := strings.Split(cfg.ToEmail, ",")
    addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

    status := "FAILURE"
    if success {
        status = "SUCCESS"
    }

    subject := fmt.Sprintf("[AIOps] Remediation Action Report: %s - %s", status, action.ActionType)
    body := fmt.Sprintf(
        "AIOps Remediation Action has been executed.\r\n\r\n"+
            "Status      : %s\r\n"+
            "Action Type : %s\r\n"+
            "Namespace   : %s\r\n"+
            "Resource    : %s\r\n"+
            "Reason      : %s\r\n"+
            "Triggered By: %s\r\n\r\n"+
            "Details:\r\n%s\r\n",
        status,
        action.ActionType,
        action.Namespace,
        action.ResourceName,
        action.Reason,
        action.TriggeredBy,
        details,
    )

    // RFC 822 형식 메시지
    msg := []byte(
        fmt.Sprintf("From: %s\r\n", cfg.FromEmail) +
            fmt.Sprintf("To: %s\r\n", strings.Join(to, ",")) +
            fmt.Sprintf("Subject: %s\r\n", subject) +
            "MIME-Version: 1.0\r\n" +
            "Content-Type: text/plain; charset=UTF-8\r\n" +
            "\r\n" +
            body,
    )

    // 1. 평문으로 SMTP 서버에 접속
    client, err := smtp.Dial(addr)
    if err != nil {
        return fmt.Errorf("failed to dial SMTP server: %w", err)
    }
    defer client.Close()

    // 2. STARTTLS로 업그레이드
    tlsConfig := &tls.Config{
        ServerName: cfg.Host,
    }
    if err := client.StartTLS(tlsConfig); err != nil {
        return fmt.Errorf("failed to start TLS: %w", err)
    }

    // 3. AUTH PLAIN (네이버: 앱 비밀번호 사용)
    auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
    if err := client.Auth(auth); err != nil {
        return fmt.Errorf("failed to authenticate: %w", err)
    }

    // 4. MAIL FROM
    if err := client.Mail(cfg.FromEmail); err != nil {
        return fmt.Errorf("failed to set MAIL FROM: %w", err)
    }

    // 5. RCPT TO (여러 수신자)
    for _, toAddr := range to {
        toAddr = strings.TrimSpace(toAddr)
        if toAddr == "" {
            continue
        }
        if err := client.Rcpt(toAddr); err != nil {
            return fmt.Errorf("failed to set RCPT TO for %s: %w", toAddr, err)
        }
    }

    // 6. DATA
    wc, err := client.Data()
    if err != nil {
        return fmt.Errorf("failed to get DATA writer: %w", err)
    }

    if _, err := wc.Write(msg); err != nil {
        _ = wc.Close()
        return fmt.Errorf("failed to write email body: %w", err)
    }
    if err := wc.Close(); err != nil {
        return fmt.Errorf("failed to close DATA writer: %w", err)
    }

    // 7. QUIT
    if err := client.Quit(); err != nil {
        return fmt.Errorf("failed to quit SMTP client: %w", err)
    }

    return nil
}
