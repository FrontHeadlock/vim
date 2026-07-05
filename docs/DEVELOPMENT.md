# VimQuest 개발 가이드

## 개발 환경 설정

### 필수 도구

**Go**
```bash
# Go 1.26+ 설치
# macOS: brew install go
# Linux: https://go.dev/doc/install
go version  # 1.26 이상 확인
```

**Ebiten (데스크톱 빌드)**
```bash
go get github.com/hajimehoshi/ebiten/v2
```

**TinyGo (웹 빌드)**
```bash
# macOS
brew tap tinygo-org/tools
brew install tinygo-org/tools/tinygo

# Linux
https://tinygo.org/getting-started/install/

# 설치 확인
tinygo version
```

---

## 프로젝트 구조

```
.
├── cmd/
│   ├── desktop/          Ebiten 프론트엔드
│   │   ├── main.go      (게임 루프, 입력)
│   │   └── render.go    (렌더링)
│   └── web/             TinyGo WASM 진입점
│       └── main.go
├── internal/
│   ├── engine/          Vim 편집 엔진 (순수 로직, 파일별 책임 분할)
│   │   ├── editor.go    (타입·Feed 디스패치)
│   │   ├── normal.go / motion.go / operator.go / ...
│   │   └── keys.go
│   ├── game/            게임 규칙 & 상태 머신
│   │   ├── game.go
│   │   ├── effects.go
│   │   ├── drill.go
│   │   ├── levels.go        (규칙 데이터: Map/Target/Solution)
│   │   ├── levels_meta.go   (표시 데이터: 제목·힌트·팔레트 — !js, wasm 제외)
│   │   ├── view.go      (데스크톱 렌더러 접근자)
│   │   └── snapshot.go  (웹 렌더러 계약)
│   ├── store/           진행 저장
│   │   ├── store.go
│   │   ├── store_js.go
│   │   └── store_other.go
│   ├── platform/        DOM/SFX 브리지
│   │   ├── dom_js.go
│   │   └── dom_other.go
│   └── jsbridge/        JS ↔ Go 연결부
│       └── bridge.go
├── test/                모든 테스트(블랙박스 — 공개 API 만 사용)
│   ├── engine/          (+ fuzz 코퍼스 testdata/)
│   ├── game/
│   └── store/
├── tools/genmeta/       levels_meta.go → levels_meta.js 생성기
├── web/
│   ├── src/             (소스 코드)
│   │   ├── index.html
│   │   ├── levels_meta.js   (생성 파일 — 커리큘럼 표시 데이터)
│   │   ├── renderer.js
│   │   └── glue.js
│   └── dist/            (빌드 산출물)
├── scripts/
│   ├── build.sh         웹 빌드
│   └── build_desktop.sh 데스크톱 빌드
├── Makefile
├── go.mod
└── docs/
    ├── ARCHITECTURE.md  아키텍처
    ├── DEVELOPMENT.md   (이 파일)
    └── screenshot.png
```

---

## 빌드 & 실행

### 전체 빌드
```bash
make              # 또는 make build
```

### 개별 빌드

**데스크톱 (Ebiten)**
```bash
make build-desktop
# 또는
./scripts/build_desktop.sh
# → ./vimquest 바이너리 생성
```

**웹 (TinyGo WASM)**
```bash
make build-web
# 또는
./scripts/build.sh
# → web/dist/game.wasm 생성
```

### 실행

**데스크톱**
```bash
./vimquest
```

**웹**
```bash
# 간단한 HTTP 서버 (Python 3)
python3 -m http.server 8765 --directory web

# 브라우저에서 접속
open http://localhost:8765/src/
```

---

## 테스트

### 모든 테스트 실행
```bash
make test
# 또는
go test ./...
```

### 특정 패키지 테스트
```bash
go test ./test/engine -v
go test ./test/game -v
go test ./test/store -v
```

### 테스트 커버리지
```bash
go test ./... -cover
```

### 특정 테스트만 실행
```bash
go test ./test/engine -run TestMotion -v
```

---

## 코드 스타일 & 검증

### Go 포맷팅
```bash
go fmt ./...
```

### Lint 검사
```bash
go vet ./...
```

### 전체 검증
```bash
go fmt ./... && go vet ./...
```

---

## 주요 파일 가이드

### game.go (433줄) — 핵심 게임 상태 머신

**구조:**
```go
type Game struct {
  levelIdx int              // 현재 레벨 인덱스
  lv Level                  // 현재 레벨
  ed *engine.Editor         // 편집 엔진
  keyPos map[[2]int]bool    // navigate 레벨의 열쇠 위치
  state State               // 현재 화면 상태
  // ... (15개 필드)
}
```

**상태 변경 메서드:**
- `Input(key)` — 키 입력 처리
- `Tick()` — 프레임 진행 (이펙트 TTL)
- `LoadLevel(idx)` — 레벨 로드
- `RestartCurrent()` — 레벨 재시작

**읽기 접근자 (view.go 참고):**
- 렌더러는 상태를 읽기만 하고 변경 불가

---

### editor.go (1602줄) — Vim 편집 엔진

**모드:**
- ModeNormal — 이동 & 명령 (기본)
- ModeInsert — 타이핑
- ModeVisual — 문자 선택
- ModeVisualLine — 줄 선택

**주요 기능:**
- 모션: h/j/k/l, w/b/e, 0/^/$, f/F/t/T, gg/G
- 연산자: d(delete), c(change), y(yank)
- 텍스트 객체: iw, aw, i", a(, 등
- 검색: /, ?, n, N
- 반복: . (dot)

**내부 상태 (비공개):**
```go
lines [][]rune        // 버퍼
row, col int          // 커서 위치
dcol int              // j/k의 목표 열
mode Mode
count int             // 명령 반복 수
op rune               // 현재 연산자 (d/c/y)
// ... (복잡한 파싱 상태)
```

**공개 API:**
```go
// 읽기
Row() int
Col() int
Lines() []string
Mode() Mode
Cell(row, col int) (rune, bool)

// 수정
SetCursor(row, col int)  // 원자적 — dcol도 함께 설정
Feed(key Key)
```

---

### effects.go (60줄) — 시각 피드백

**Effect 구조:**
```go
type Effect struct {
  Row, Col int
  Glyph rune          // 표시할 문자 (0이면 치환 없음)
  Invert bool         // 반전 여부
  TTL int             // 남은 프레임
}
```

**이벤트 → 연출 + 사운드:**
- "key" — 열쇠 획득: 반전, 핑 음
- "bug" — 버그 처치: 'x' 표시, 낮은 음
- "blocked" — 막힌 키: visual bell
- "clear" — 레벨 클리어: 상승 음 3개

---

### drill.go (127줄) — :drill 절차 생성

**특징:**
- 무작위 navigate 그리드 생성 (5×20)
- 그리디 해(hjkl만 사용) 자동 생성 + 검증
- 세션 한정 (진행 저장 안 함)

**generateDrill():**
```
1. 모든 칸(100개) 무작위 배열
2. 칸 3개 선택: 시작(@), 종료($), 열쇠(1~3개, K)
3. 경로 계산: 각 열쇠 → 종료로, hjkl만 사용
4. Solution에 저장
```

---

### levels.go (646줄) — 레벨 정의 & 세계 구조

**레벨 구조:**
```go
type Level struct {
  ID       string         // "L1.1", "L2.3" 등
  Kind     string         // "navigate" 또는 "edit"
  Title    string
  Hint     string
  Map      []string       // 그리드
  Solution string         // hjkl 시퀀스
  Par      int            // 최적 타수
}
```

**worldGroups():**
- 레벨을 월드별로 그룹화
- 각 월드: 기본 문제 + 보강 L3
- 레벨 선택 화면의 커서 네비게이션

---

### store (3개 파일) — 진행 저장

**저장 형식:**
```go
type LevelProgress struct {
  BestStrokes int
  Stars       int
  Unlocked    bool
}
```

**인코딩:**
```
수제 텍스트 코덱("1-1:1,13,3;3-2:1,0,2") → localStorage/파일
```
encoding/json 대신 손으로 짠 이유: TinyGo wasm 빌드에서 reflection 비용이 커서
(89KB→458KB 실측) 레벨당 3필드뿐인 이 스키마엔 맞지 않는다(store.go 주석 참고).

**구현:**
- `store_js.go` — 웹: localStorage (크롬 개발자 도구에서 확인 가능)
- `store_other.go` — 데스크톱: `os.UserConfigDir()/vimquest/progress.txt`
  (macOS: `~/Library/Application Support/vimquest/progress.txt`). `go test` 하에서는
  `testing.Testing()` 감지로 인메모리 구현으로 자동 대체된다.

---

### view.go (64줄) — 데스크톱 렌더러 접근자

**목적:** game의 내부 상태를 읽기 전용으로 노출

**접근자:**
```go
State() State                             // 현재 화면 (Playing/Clear/LevelSelect 등)
Editor() *engine.Editor                   // 버퍼 & 커서
Level() Level                             // 현재 레벨
LevelIndex() int                          // 레벨 인덱스
Strokes() int                             // 타수
Par() int                                 // 최적 타수
ProgressFor(id string) store.LevelProgress // 저장된 진행
// ... (30+ 접근자)
```

**특징:**
- 상태 변경 불가 (읽기 전용)
- 렌더 로직은 cmd/desktop/render.go에서 사용

---

### snapshot.go (125줄) — 웹 렌더러 계약

**목적:** 게임 상태를 JS에 전달 가능한 맵으로 변환

**구조:**
```go
func (g *Game) Snapshot() map[string]any {
  return map[string]any{
    "state": int(g.state),
    "levelIdx": g.levelIdx,
    "lines": lines,
    "row": g.ed.Row(),
    "col": g.ed.Col(),
    // ...
  }
}
```

**웹에서 JS로:**
- JSON 직렬화 → glue.js → renderer.js → canvas 렌더링

---

## 일반적인 개발 작업

### 새 레벨 추가

1. `internal/game/levels.go`에서 `levels` 배열에 추가:
```go
{
  ID: "L1.2",
  Kind: "navigate",
  Title: "Move",
  Map: []string{
    ".K.$",
    ".....",
  },
  Solution: "ljl",
  Par: 3,
}
```

2. 빌드 & 테스트:
```bash
make build
./vimquest  # 또는 make build-web && 웹 테스트
```

### 새 Vim 기능 추가

1. `internal/engine/editor.go`의 `feed()` 메서드 수정
2. 테스트 추가 (`editor_test.go`)
3. 빌드 & 테스트:
```bash
go test ./test/engine -v
make build
```

### 게임 규칙 변경

1. `internal/game/game.go` 또는 관련 파일 수정
2. `internal/game/game_test.go`에 테스트 추가
3. 빌드 & 테스트:
```bash
go test ./test/game -v
make build
```

### UI 변경

**데스크톱:**
- `cmd/desktop/render.go` 수정
- `make build-desktop && ./vimquest`

**웹:**
- `web/src/index.html` (레이아웃)
- `web/src/renderer.js` (canvas 렌더링)
- `web/src/glue.js` (JS ↔ Go 접착제)
- `make build-web && 웹 서버 재시작`

---

## 성능 최적화

### 데스크톱 (Ebiten)
- 렌더링은 `render.go`의 `Draw()` 메서드에서만
- 상태 읽기는 `view.go` 접근자 사용 (캐싱 기회 있음)

### 웹 (TinyGo)
- GC 비활성화 (-gc=leaking)
- 크기 예산: 125KB gzip
- 메모리 누수 주의 (쓰레기 회수 안 함)

### 프로파일링

**Go:**
```bash
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

---

## 문제 해결

### "tinygo: command not found"
```bash
brew install tinygo-org/tools/tinygo
```

### 웹 빌드 크기 초과
```
✗ 크기 예산 초과: 130KB > 125KB
```

해결:
- 표준 라이브러리 임포트 최소화
- 불필요한 함수/타입 제거
- refactor.md의 선택지 참고

### 테스트 실패
```bash
go test ./... -v
# 실패한 테스트만
go test ./internal/game -run TestName -v
```

---

## 커밋 & PR

### 커밋 메시지 포맷
```
type: 주제

상세 설명 (필요시)

Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>
```

**type:**
- `feat:` 새 기능
- `fix:` 버그 수정
- `refactor:` 코드 재구성
- `docs:` 문서
- `test:` 테스트

### PR 체크리스트
- [ ] `make build` 성공
- [ ] `make test` 통과 (전부)
- [ ] `go vet` 클린
- [ ] 데스크톱 / 웹 양쪽 수동 테스트
- [ ] 웹 크기 예산 <= 125KB gzip

---

## 참고

- **아키텍처:** docs/ARCHITECTURE.md
- **refactor.md:** 패키지 분할 히스토리 & TinyGo 크기 분석
- **ROADMAP.md:** 프로젝트 로드맵
