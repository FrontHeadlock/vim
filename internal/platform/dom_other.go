//go:build !js

package platform

// 데스크톱 빌드에서는 DOM 도 JS 신스도 없으므로 전부 no-op.
// (Ebiten 으로 .app/.exe 빌드 시 컴파일이 깨지지 않도록 유지)

func SetText(id, value string) {}

func SetHTML(id, html string) {}

func ShowOverlay(id string) {}

func Sfx(name string) {}
