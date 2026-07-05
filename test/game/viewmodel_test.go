package gametest

import (
	"testing"

	. "vimquest/internal/game"
)

// TestVisualRowsCharwiseSingleLine 은 같은 줄 charwise 선택이 [c1,c2] 구간
// 하나로 정확히 나오는지 확인한다.
func TestVisualRowsCharwiseSingleLine(t *testing.T) {
	g := New()
	g.LoadCustomLevel(Level{Kind: "edit", Map: []string{"hello world"}, Target: []string{"hello worldX"}})
	playKeys(g, "llv ll") // v 로 진입 후 이동 선택
	rows := g.VisualRows()
	if len(rows) != 1 {
		t.Fatalf("len(rows)=%d want 1 (got %+v)", len(rows), rows)
	}
	if rows[0].Row != 0 {
		t.Fatalf("row=%d want 0", rows[0].Row)
	}
}

// TestVisualRowsLinewiseSpansWholeLines 은 V(라인 선택) 로 여러 줄을 고르면
// 각 줄 전체가 구간(0..len-1)으로 나오는지 확인한다.
func TestVisualRowsLinewiseSpansWholeLines(t *testing.T) {
	g := New()
	g.LoadCustomLevel(Level{Kind: "edit", Map: []string{"aa", "bbb", "c"}, Target: []string{"aa", "bbb", "cX"}})
	playKeys(g, "Vj")
	rows := g.VisualRows()
	if len(rows) != 2 {
		t.Fatalf("len(rows)=%d want 2 (got %+v)", len(rows), rows)
	}
	if rows[0].Row != 0 || rows[0].C1 != 0 || rows[0].C2 != 1 {
		t.Errorf("row0=%+v want {0,0,1}", rows[0])
	}
	if rows[1].Row != 1 || rows[1].C1 != 0 || rows[1].C2 != 2 {
		t.Errorf("row1=%+v want {1,0,2}", rows[1])
	}
}

// TestVisualRowsNilWhenNotSelecting 는 비주얼 모드가 아니면 nil 인지 확인한다.
func TestVisualRowsNilWhenNotSelecting(t *testing.T) {
	g := New()
	g.LoadCustomLevel(Level{Kind: "edit", Map: []string{"abc"}, Target: []string{"abcX"}})
	if rows := g.VisualRows(); rows != nil {
		t.Fatalf("비선택 상태인데 VisualRows()=%+v want nil", rows)
	}
}

// TestMatchedRowsPerLine 은 edit 레벨에서 각 줄이 Target 과 일치하는지를
// 정확히 반영하는지 확인한다.
func TestMatchedRowsPerLine(t *testing.T) {
	g := New()
	g.LoadCustomLevel(Level{
		Kind:   "edit",
		Map:    []string{"foo", "bar"},
		Target: []string{"foo", "baz"},
	})
	matched := g.MatchedRows()
	if len(matched) != 2 || !matched[0] || matched[1] {
		t.Fatalf("matched=%+v want [true false]", matched)
	}
}

// TestMatchedRowsNilForNavigate 는 navigate 레벨에서 nil 인지 확인한다
// (edit 전용 계약 — navigate 렌더 경로는 이 필드를 아예 쓰지 않는다).
func TestMatchedRowsNilForNavigate(t *testing.T) {
	g := New()
	if got := g.MatchedRows(); got != nil {
		t.Fatalf("navigate 레벨인데 MatchedRows()=%+v want nil", got)
	}
}

// TestSnapshotContract 는 각 화면 상태에서 Snapshot() 이 렌더러가 반드시
// 필요로 하는 키를 빠짐없이 채우는지 확인한다 — 스냅샷 필드가 늘어날 때
// 문서화 없이 조용히 계약이 깨지는 것을 잡는다. 각 상태는 실제 플레이
// 흐름(레벨 클리어·선택 진입 등)으로 도달한다.
func TestSnapshotContract(t *testing.T) {
	requireKeys := func(t *testing.T, snap map[string]any, keys ...string) {
		t.Helper()
		for _, k := range keys {
			if _, ok := snap[k]; !ok {
				t.Errorf("state=%v: 스냅샷에 %q 키 없음", snap["state"], k)
			}
		}
	}

	// playing (navigate)
	g := New()
	requireKeys(t, g.Snapshot(), "state", "kind", "lines", "row", "col", "mode",
		"strokes", "par", "visualRows", "hint", "title")

	// playing (edit)
	for i := 0; i < LevelCount(); i++ {
		if LevelAt(i).Kind == "edit" {
			g.LoadLevel(i)
			break
		}
	}
	requireKeys(t, g.Snapshot(), "target", "matchedRows")

	// clear — 1-1 을 실제로 클리어해 도달
	g2 := New()
	playKeys(g2, "jjlllljjlllll")
	requireKeys(t, g2.Snapshot(), "state", "clearStrokes", "clearPar", "clearStars",
		"clearBest", "clearIsNew", "clearYours", "solution")

	// select
	g3 := New()
	g3.EnterLevelSelect()
	requireKeys(t, g3.Snapshot(), "state", "worlds", "selRow", "selCol")

	// allclear — 마지막 레벨을 실제로 클리어해 도달
	g4 := reachAllClear(t)
	requireKeys(t, g4.Snapshot(), "state", "worldCount", "levelCount")
}
