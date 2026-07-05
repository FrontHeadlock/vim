#!/usr/bin/env bash
# VimQuest 데스크톱 빌드 스크립트 — Ebiten 프론트엔드(main.go, //go:build !js)를
# 네이티브 바이너리로 컴파일한다. 웹 빌드는 build.sh(TinyGo) 를 쓸 것.
set -euo pipefail

cd "$(dirname "$0")"

echo "▶ 데스크톱 빌드 중..."
go build -o vimquest .

echo "✅ 빌드 완료 → ./vimquest"
