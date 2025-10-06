package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opslinked-ai/remediation-module/pkg/config"
	"opslinked-ai/remediation-module/pkg/handlers"
	"opslinked-ai/remediation-module/pkg/k8s"
	"opslinked-ai/remediation-module/pkg/logging"

	"go.uber.org/zap"
)

func main() {
	// 1. Viper를 사용한 설정 초기화
	cfg, err := config.LoadConfig(".")
	if err!= nil {
		panic(fmt.Errorf("cannot load config: %w", err))
	}

	// 2. Zap을 사용한 구조화된 로거 설정
	logger := logging.InitLogger(cfg.Log.Level)
	defer logger.Sync()

	logger.Info("Starting Remediation Module", zap.String("logLevel", cfg.Log.Level))

	// 4. In-Cluster Kubernetes 클라이언트 초기화
	k8sClient, err := k8s.NewK8sClient()
	if err!= nil {
		logger.Fatal("Failed to create Kubernetes client", zap.Error(err))
	}
	logger.Info("Successfully initialized Kubernetes client")

	// 5. HTTP 라우터 설정
	mux := http.NewServeMux()
	remediationHandler := handlers.NewRemediationHandler(k8sClient, cfg, logger)
	mux.HandleFunc("/remediate", remediationHandler.Handle)

	// 3. Graceful Shutdown을 위한 HTTP 서버 설정
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 6. HTTP 서버 실행 (고루틴)
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server starting", zap.Int("port", cfg.Server.Port))
		if err := server.ListenAndServe();!errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Graceful shutdown 로직
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Fatal("Server error", zap.Error(err))
	case sig := <-quit:
		logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
	}

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err!= nil {
		logger.Fatal("Server shutdown failed", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}