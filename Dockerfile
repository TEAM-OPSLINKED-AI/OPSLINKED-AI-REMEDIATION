# --- Builder Stage ---
# Go 컴파일을 위한 빌더 이미지
FROM golang:1.22-alpine AS builder

# 작업 디렉토리 설정
WORKDIR /app

# Go 모듈 의존성 다운로드
COPY go.mod go.sum ./
RUN go mod download

# 소스 코드 복사
COPY . .

# 애플리케이션 빌드
# CGO_ENABLED=0: C 의존성 없이 정적 바이너리 생성
# GOOS=linux: 리눅스 환경용으로 빌드
# -ldflags "-w -s": 디버깅 정보 제거하여 바이너리 크기 최적화
# Changed: 빌드 경로를 현재 디렉터리를 의미하는 '.'으로 수정
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -s' -o /remediation-server .

# --- Final Stage ---
# 최종 이미지를 위한 최소한의 베이스 이미지
FROM scratch

# 루트 CA 인증서 복사 (HTTPS/TLS 통신에 필요)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 빌더 스테이지에서 컴파일된 실행 파일 복사
COPY --from=builder /remediation-server /remediation-server

# 애플리케이션 실행 포트 노출
EXPOSE 8080

# 컨테이너 시작 시 실행될 명령어
ENTRYPOINT ["/remediation-server"]