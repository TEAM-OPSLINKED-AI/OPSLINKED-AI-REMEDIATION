package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"opslinked-ai/remediation-module/pkg/config"
	"opslinked-ai/remediation-module/pkg/k8s"
	"opslinked-ai/remediation-module/pkg/notification"
	"opslinked-ai/remediation-module/pkg/types" // types 패키지 import

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

// RemediationAction 구조체 정의를 이 파일에서 삭제합니다.

type RemediationHandler struct {
	K8sClient *kubernetes.Clientset
	Config    config.Config
	Logger    *zap.Logger
}

func NewRemediationHandler(client *kubernetes.Clientset, cfg config.Config, logger *zap.Logger) *RemediationHandler {
	return &RemediationHandler{
		K8sClient: client,
		Config:    cfg,
		Logger:    logger,
	}
}

func (h *RemediationHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var action types.RemediationAction // types.RemediationAction 사용
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		h.Logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if action.ActionType == "" || action.ResourceName == "" {
		h.Logger.Error("Validation failed: actionType and resourceName are required")
		http.Error(w, "actionType and resourceName are required fields", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Remediation action accepted for asynchronous processing."))

	go h.handleAction(action)
}

func (h *RemediationHandler) handleAction(action types.RemediationAction) { // types.RemediationAction 사용
    startTime := time.Now()
	logger := h.Logger.With(
		zap.String("actionType", action.ActionType),
		zap.String("namespace", action.Namespace),
		zap.String("resourceName", action.ResourceName),
	)

	logger.Info("Starting to handle remediation action")

	var success bool
	var details string
	var err error

	switch action.ActionType {
	case "RESTART_DEPLOYMENT":
		err = k8s.RestartDeployment(h.K8sClient, action.Namespace, action.ResourceName)
		if err == nil {
			details = fmt.Sprintf("Successfully triggered rolling restart for Deployment '%s/%s'.", action.Namespace, action.ResourceName)
		}
	case "EXECUTE_NODE_SHELL_COMMAND":
		details, err = k8s.ExecuteNodeShellCommand(h.K8sClient, action.ResourceName, action.Parameters)
	case "ALERT_ONLY":
		usage := action.Parameters["usagePercentage"]
		details = fmt.Sprintf("High-level alert for resource '%s/%s'. Current usage: %s%%. No automated action was taken.", action.Namespace, action.ResourceName, usage)
		err = nil
	default:
		err = fmt.Errorf("unknown actionType: %s", action.ActionType)
	}

	if err != nil {
		success = false
		details = fmt.Sprintf("Failed to execute action. Error: %v", err)
		logger.Error("Action execution failed", zap.Error(err), zap.Duration("duration", time.Since(startTime)))
	} else {
		success = true
		logger.Info("Action execution successful", zap.String("details", details), zap.Duration("duration", time.Since(startTime)))
	}

	recipients := strings.Split(h.Config.SMTP.ToEmail, ",")
    // 빈 문자열 슬라이스가 아닌지 확인
	if len(recipients) > 0 && recipients[0] != "" {
		err = notification.SendEmailNotification(h.Config.SMTP, action, success, details)
		if err != nil {
			logger.Error("Failed to send email notification", zap.Error(err))
		} else {
			logger.Info("Successfully sent email notification")
		}
	} else {
		logger.Warn("SMTP 'toEmail' is not configured. Skipping notification.")
	}
}