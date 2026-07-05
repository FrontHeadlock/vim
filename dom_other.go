//go:build !js

package main

// 데스크톱 빌드에서는 DOM 이 없으므로 no-op.
// (Ebiten 으로 .app/.exe 빌드 시 컴파일이 깨지지 않도록 유지)
func domSet(id, value string) {}

func domSetHTML(id, html string) {}

func domShow(id string) {}

func jsSfx(name string) {}
