# 의존성 추가가
go mod tidy

cd ./cmd/remediation-server

# 코드 빌드
go build -o opslinked-ai-remediation
# linker 오류시 ldflags 플래그로 빌드
go build -ldflags="-extldflags=-lssp" -o opslinked-ai-remediation

# 실행 파일 실행
./opslinked-ai-remediation

# main.go 실행
go run -ldflags="-extldflags=-lssp" main.go

docker login

# Docker 이미지 빌드
docker build -t judemin/opslinked-ai-remediation:1.0 .

# Docker 이미지 Push
docker push judemin/opslinked-ai-remediation:1.0