# VimQuest 아키텍처

## 개요

VimQuest는 패키지 분할 구조로 설계되어 있으며, 의존 방향이 **단방향**으로 강제된다:

```
cmd/* → game → {engine, store, platform}
```

이는 컴파일러 수준에서 계층 경계를 강제하고, 테스트 용이성과 웹 빌드(TinyGo) 예산 관리를 가능하게 한다.

---

## 디렉토리 구조

### `cmd/` — 진입점 & 프론트엔드

두 개의 플랫폼별 진입점:

#### `cmd/desktop/` — Ebiten 데스크톱 프론트엔드
- **main.go** — 게임 루프, 입력 폴링, Ebiten 초기화
- **render.go** — 화면 렌더링 (터미널 에뮬레이션)
- 의존: `game`, `engine` (view 접근자 통해)
- 빌드: `./scripts/build_desktop.sh`

#### `cmd/web/` — TinyGo WebAssembly 진입점
- **main.go** — JS 브리지, WASM 콜백 (vqInput/vqTick)
- 의존: `game`, `jsbridge`
- 빌드: `./scripts/build.sh` (TinyGo)

---

### `internal/` — 게임 로직 & 엔진

#### `internal/engine/` — 순수 Vim 편집 엔진

**의존성:** 표준 라이브러리만 (외부 디펜던시 없음)

**파일:**
- **editor.go** (~1600줄) — 핵심 편집 엔진
  - Normal/Insert/Visual 모드
  - 모션, 연산자, 텍스트 객체, 검색
  - Undo/Redo, 반복(dot)

- **keys.go** — 키 토크나이저 (hjkl, f/t, 명령 파싱)

- **editor_test.go** — 48개 테스트

**공개 API:**
```go
type Mode int
type Key struct { R rune; S string }

// 생성
NewEditor(lines []string) *Editor

// 읽기 전용
Row() int
Col() int
Lines() []string
Mode() Mode
Searching() bool
MidCommand() bool
IsCmdStart() bool
Cell(row, col int) (rune, bool)
PendingString() string
LastKey() string

// 상태 변경
SetCursor(row, col int)      // row/col/dcol 원자적 설정
Feed(key Key)
GotoLine(n int)
ParseKeys(s string) []Key
```

---

#### `internal/game/` — 게임 규칙 & 상태 머신

**의존성:** `engine`, `store`, `platform`

**파일:**
- **game.go** — 메인 상태 머신 & 라이프사이클
  - 레벨 진행, 별점 계산, 클리어 판정
  - Input/Tick/LoadLevel/RestartCurrent 진입점

- **effects.go** — 터미널식 피드백 연출
  - 문자 치환, 반전, visual bell
  - 이펙트 TTL 관리

- **drill.go** — :drill 절차 생성 연습 모드
  - 무작위 navigate 문제 생성
  - 그리디 해(solution) 자동 검증

- **levels.go** — 레벨 정의 & worldGroups
  - 모든 레벨 데이터
  - 월드별 그룹핑 & 이전 레벨 연쇄 해금

- **view.go** — 데스크톱 렌더러용 읽기 전용 접근자
  - `State()`, `Editor()`, `Level()`, `Strokes()`, `Par()` 등
  - 렌더러가 상태를 조회만 하고 변경 불가

- **snapshot.go** — 웹 렌더러용 데이터 계약
  - `Snapshot() map[string]any` — JS로 serialize 가능한 형식

- **dom.go** — UI 동기화 헬퍼
  - 한국어 명령 조회 (ex-command 힌트)

- **game_test.go** — 게임 규칙 테스트

**상태 변경 진입점:**
```go
Input(key engine.Key)        // 키 입력
Tick()                        // 프레임 진행 (이펙트 TTL 감소)
LoadLevel(idx int)            // 레벨 로드
RestartCurrent()              // 현재 레벨 재시작
EnterLevelSelect()            // 레벨 선택 화면
```

**읽기 전용 접근자 (view.go):**
```go
State() State
Editor() *engine.Editor
Level() Level
LevelIndex() int
Strokes() int
KeysLeft() int
ProgressFor(id string) store.LevelProgress
// ... 30+ 접근자
```

---

#### `internal/store/` — 진행 저장

**의존성:** 표준 라이브러리 (선택적 syscall/js)

**파일:**
- **store.go** — 저장소 인터페이스 & 코덱
  - LevelProgress (BestStrokes, Stars, Unlocked)
  - gob + base32 인코딩

- **store_js.go** — 웹 구현 (localStorage)
- **store_other.go** — 데스크톱 구현 (홈 디렉토리)
- **store_test.go** — 코덱 테스트

**공개 API:**
```go
type Store interface {
  Load() map[string]LevelProgress
  Save(p map[string]LevelProgress)
}

func New() Store  // 빌드 태그로 구현 선택
```

---

#### `internal/platform/` — DOM/SFX 브리지

**의존성:** 선택적 syscall/js (데스크톱은 no-op)

**파일:**
- **dom_js.go** — 웹 구현 (JS 호출)
  - `SetText(id, text)` — HTML 요소 업데이트
  - `SetHTML(id, html)` — HTML 설정
  - `ShowOverlay(id)` — 모달 표시

- **dom_other.go** — 데스크톱 구현 (no-op)

**공개 API:**
```go
SetText(id, text string)
SetHTML(id, html string)
ShowOverlay(id string)
Sfx(name string)  // "key", "bug", "blocked", "clear"
```

---

#### `internal/jsbridge/` — JS/게임 연결부

**파일:**
- **bridge.go** — JS 콜백 등록
  - `vqInput(keyCode, keyName)` — JS → Go
  - `vqState()` → 게임 상태
  - `vqTick()` — 매 프레임 호출

---

### `web/` — 정적 자산

```
web/
├── src/          (소스 코드)
│   ├── index.html       (진입점, 레이아웃)
│   ├── renderer.js      (canvas 2D 렌더러)
│   └── glue.js          (JS ↔ Go 브리지)
└── dist/         (빌드 산출물)
    ├── game.wasm        (TinyGo 컴파일 결과)
    └── wasm_exec.js     (TinyGo 런타임)
```

---

## 의존 방향 & 경계 강제

### 원칙
- **game** ← 모든 패키지가 의존 가능
- **engine/store/platform** ← game만 의존 가능
- **cmd/** ← 진입점(no-op), 의존성 역방향 금지
- **ebiten/syscall-js** ← cmd 제외 import 금지

### 컴파일 수준 강제
Go의 패키지 구조로 비순환 의존성(DAG)이 자동 검증된다.
대부분의 계층 위반은 컴파일 에러로 즉시 적발된다.

### 테스트 분리
- `internal/engine` — 순수 로직, headless 테스트만으로 충분
- `internal/game` — 게임 규칙, 게임 상태 테스트
- 통합 테스트는 끝-끝 플레이(web, desktop)로만 수행

---

## 주요 설계 결정

### 1. 읽기 전용 뷰 패턴 (view.go)

**문제:** game.go의 내부 필드(ed.row, ed.col)가 직접 수정되면 dcol(j/k 목표열) 갱신 누락 → 커서 점프 버그

**해결:** 
- Editor 내부 필드 전부 비공개
- SetCursor()로만 설정 (원자적)
- 렌더러는 Row()/Col() 읽기만 가능

### 2. 분리된 웹/데스크톱 렌더러

**문제:** 같은 game 상태를 두 가지 방식으로 렌더링
- 데스크톱: 인메모리 터미널 에뮬레이션 (Ebiten 픽셀링)
- 웹: JSON/canvas (JS)

**해결:**
- desktop: view.go 접근자 (읽기 + 로컬 렌더링)
- web: snapshot.go (게임 → JSON → JS)

### 3. TinyGo 크기 예산 (105KB gzip)

**제약:** WebAssembly 페이로드 최소화
- 패키지당 DCE/메타데이터 오버헤드 ~4KB
- 단순 패키지 분할 시 124KB → 105KB로 상향

**대책:**
- GC 비활성화 (-gc=leaking)
- 표준 라이브러리 최소 사용
- 내부 타입 전방위 공개 제한

---

## 빌드

### Makefile
```bash
make              # 전체 빌드
make build-web    # 웹 (TinyGo)
make build-desktop # 데스크톱 (Ebiten)
make test         # 테스트
make clean        # 산출물 정리
```

### 직접 실행
```bash
./scripts/build.sh           # 웹
./scripts/build_desktop.sh   # 데스크톱
```

---

## 시작하기

### 요구사항
- Go 1.26+
- Ebiten (데스크톱): `go get github.com/hajimehoshi/ebiten/v2`
- TinyGo (웹): `brew install tinygo`

### 빌드
```bash
make build
```

### 실행
```bash
# 데스크톱
./vimquest

# 웹
python3 -m http.server 8765 --directory web
# → http://localhost:8765/src/
```
