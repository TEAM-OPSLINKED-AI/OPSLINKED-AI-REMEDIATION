# --- Builder Stage ---
    FROM golang:1.22-alpine AS builder

    WORKDIR /app
    
    COPY go.mod go.sum ./
    RUN go mod download
    
    COPY . .
    
    RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -s' -o /remediation-server .
    
    # --- Final Stage ---
    # Changed: 'scratch'에서 'alpine'으로 변경
    FROM alpine:3.20
    
    # 루트 CA 인증서 복사 (기존과 동일)
    COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
    
    # 빌더 스테이지에서 컴파일된 실행 파일 복사 (기존과 동일)
    COPY --from=builder /remediation-server /remediation-server
    
    # --- (Changed) Scenario 2 시뮬레이션을 위한 도구 및 파일 추가 ---
    
    # 1. gzip, logrotate 설치
    #    (logrotate 패키지 설치 시 /etc/logrotate.conf가 자동으로 생성됨)
    RUN apk add --no-cache gzip logrotate
    
    # 2. 임의의 테스트 로그 파일 생성 (아래 2번 항목과 연관)
    RUN mkdir -p /var/log/test-app
    RUN echo "This is line 1 of the test log." > /var/log/test-app/app.log
    RUN echo "This is line 2 of the test log." >> /var/log/test-app/app.log
    RUN echo "This is line 3 of the test log." >> /var/log/test-app/app.log
    
    # ----------------------------------------------------------------
    
    # 애플리케이션 실행 포트 노출 (기존과 동일)
    EXPOSE 8080
    
    # 컨테이너 시작 시 실행될 명령어 (기존과 동일)
    ENTRYPOINT ["/remediation-server"]