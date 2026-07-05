.PHONY: build build-web build-desktop test test-v clean help

# 기본 타겟
all: build

# 전체 빌드 (웹 + 데스크톱)
build: build-web build-desktop
	@echo "✅ 전체 빌드 완료"

# 웹 빌드 (TinyGo WASM)
build-web:
	@./scripts/build.sh

# 데스크톱 빌드 (Ebiten)
build-desktop:
	@./scripts/build_desktop.sh

# 테스트 실행 (요약만 — 패키지별 결과 한 줄씩, E3)
test:
	@go test ./...

# 테스트 실행 (상세 — 개별 테스트 이름까지 전부 출력)
test-v:
	@go test ./... -v

# 빌드 산출물 정리
clean:
	@echo "▶ 빌드 산출물 정리 중..."
	@rm -f vimquest
	@rm -rf web/dist/*
	@echo "✅ 정리 완료"

# 도움말
help:
	@echo "VimQuest 빌드 명령어:"
	@echo ""
	@echo "  make              전체 빌드 (웹 + 데스크톱)"
	@echo "  make build        전체 빌드"
	@echo "  make build-web    웹 빌드 (TinyGo WASM)"
	@echo "  make build-desktop 데스크톱 빌드 (Ebiten)"
	@echo "  make test         테스트 실행 (요약)"
	@echo "  make test-v       테스트 실행 (상세, -v)"
	@echo "  make clean        빌드 산출물 정리"
	@echo "  make help         이 도움말 표시"
