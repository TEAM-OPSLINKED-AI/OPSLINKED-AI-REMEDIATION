package notification

import (
	"fmt"
	"net/smtp"
	"strings"

	"opslinked-ai/remediation-module/pkg/config"
	"opslinked-ai/remediation-module/pkg/types" // types 패키지 import
)

// Changed: 파라미터 타입을 types.RemediationAction으로 변경
func SendEmailNotification(cfg config.SMTPConfig, action types.RemediationAction, success bool, details string) error {
	if cfg.Host == "" || cfg.ToEmail == "" || cfg.FromEmail == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	to := strings.Split(cfg.ToEmail, ",")

	var status string
	if success {
		status = "SUCCESS"
	} else {
		status = "FAILURE"
	}

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

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", strings.Join(to, ","), cfg.FromEmail, subject, body))

	return smtp.SendMail(addr, auth, cfg.FromEmail, to, msg)
}