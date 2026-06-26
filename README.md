# VimQuest 🎮

게임으로 배우는 Vim — 키를 모으고 텍스트 세계를 탐험하며 자연스럽게 Vim 명령어를 익히는 **무료·한국어 브라우저 RPG**.

> 기존 Vim 학습 게임(VIM Adventures 등)은 유료·영어 전용이고 한글 IME 상태에서 작동하지 않습니다.
> VimQuest 는 **무료 + 한국어 우선 + 한글 IME 자동 감지**로 입문자가 재밌게 처음 배우도록 만드는 것을 목표로 합니다.

## 컨셉

게임 보드가 곧 **Vim 버퍼**입니다. 커서가 실제 Vim처럼 텍스트 위를 움직이고, 잘하게 될수록 = Vim을 잘하게 됩니다.

| Vim 키 | 게임 동작 |
|--------|-----------|
| `h j k l` | 4방향 한 칸 이동 |
| `w b e` | 단어 단위 점프 |
| `x` | 커서 아래 글자 삭제(버그 제거) |
| `r` | 현재 레벨 리셋 |

목표: **열쇠(K)를 모으고 버그(\*)를 제거한 뒤 출구($)로** 도달.

## 실행

```bash
./build.sh                       # Go → WebAssembly 빌드
cd web && python3 -m http.server 8765
# 브라우저에서 http://localhost:8765/ 접속
```

> ⚠️ 키 입력은 **영문 입력 상태**에서만 동작합니다. 한글 모드면 경고 배너가 표시됩니다.

## 기술 스택

- **Go + [Ebitengine](https://ebitengine.org/)** → WebAssembly (단일 코드베이스로 웹/데스크톱)
- 게임 보드는 캔버스(ASCII)로, 한국어 UI 는 HTML/DOM 으로 분리 → WASM 번들 경량화 + 한글 깨짐 방지

## 구조

```
main.go        게임 루프·입력·렌더링 (Ebiten)
levels.go      레벨 데이터 (텍스트 맵)
dom_js.go      WASM 용 DOM 브리지 (한국어 UI 전달)
dom_other.go   데스크톱 빌드용 no-op
main_test.go   핵심 로직 유닛 테스트
web/           index.html · wasm_exec.js · (game.wasm 은 빌드 산출물)
GAME_DESIGN.md 게임 기획서
```

## 로드맵

현재는 **MVP**(W1 "이동의 숲" 3레벨)입니다. 다음 단계는 `GAME_DESIGN.md` 의 마일스톤 참고.
