// Package arenatest 는 Arena 리더보드 백엔드의 블랙박스 테스트다 —
// OpenDB(":memory:") + httptest 로 실제 HTTP 경계 그대로 검증한다.
package arenatest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"vimquest/internal/arena"
)

// newServer 는 인메모리 DB 를 물린 테스트 서버를 만든다.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := arena.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	srv := httptest.NewServer(arena.NewHandler(db))
	t.Cleanup(srv.Close)
	return srv
}

// submit 은 점수를 제출하고 (상태코드, 응답 바디) 를 돌려준다.
func submit(t *testing.T, srv *httptest.Server, body string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Post(srv.URL+"/api/arena/score", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST score: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.StatusCode, out
}

// leaderboard 는 리더보드를 조회해 scores 배열을 돌려준다.
func leaderboard(t *testing.T, srv *httptest.Server, query string) []any {
	t.Helper()
	resp, err := http.Get(srv.URL + "/api/arena/leaderboard" + query)
	if err != nil {
		t.Fatalf("GET leaderboard: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("leaderboard status=%d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	scores, ok := out["scores"].([]any)
	if !ok {
		t.Fatalf("scores 필드 없음: %v", out)
	}
	return scores
}

// TestSubmitAndLeaderboard 는 제출→리더보드 반영과 순위(빠른 시간 우선)를
// 확인한다.
func TestSubmitAndLeaderboard(t *testing.T) {
	srv := newServer(t)

	code, out := submit(t, srv, `{"id":"alice","ms":42000}`)
	if code != http.StatusOK || out["ok"] != true {
		t.Fatalf("submit: code=%d out=%v", code, out)
	}
	if out["best_ms"] != float64(42000) || out["rank"] != float64(1) {
		t.Errorf("첫 제출 best/rank=%v/%v, want 42000/1", out["best_ms"], out["rank"])
	}

	submit(t, srv, `{"id":"bob","ms":30000}`)
	scores := leaderboard(t, srv, "")
	if len(scores) != 2 {
		t.Fatalf("리더보드 %d명, want 2", len(scores))
	}
	first := scores[0].(map[string]any)
	if first["id"] != "bob" || first["rank"] != float64(1) || first["ms"] != float64(30000) {
		t.Errorf("1위=%v, want bob/1/30000", first)
	}
}

// TestBestOnlyKept 는 같은 ID 의 더 나쁜 시간이 베스트를 덮어쓰지 않고,
// 더 좋은 시간만 갱신됨을 확인한다.
func TestBestOnlyKept(t *testing.T) {
	srv := newServer(t)

	submit(t, srv, `{"id":"alice","ms":42000}`)
	_, out := submit(t, srv, `{"id":"alice","ms":99000}`) // 더 나쁨 — 무시
	if out["best_ms"] != float64(42000) {
		t.Errorf("나쁜 시간 제출 후 best=%v, want 42000 유지", out["best_ms"])
	}
	_, out = submit(t, srv, `{"id":"alice","ms":35000}`) // 개선
	if out["best_ms"] != float64(35000) {
		t.Errorf("개선 제출 후 best=%v, want 35000", out["best_ms"])
	}
	if n := len(leaderboard(t, srv, "")); n != 1 {
		t.Errorf("같은 ID 3회 제출 후 리더보드 %d명, want 1", n)
	}
}

// TestTiedTimesShareRank 는 동률이 competition ranking(같은 순위, 다음 순위
// 건너뜀)으로 표시되고, 제출 응답의 rank 와 리더보드의 rank 가 같은 정의를
// 쓰는지 확인한다 — 두 정의가 갈리면 방금 제출한 순위와 표의 순위가 어긋난다.
func TestTiedTimesShareRank(t *testing.T) {
	srv := newServer(t)
	submit(t, srv, `{"id":"a","ms":30000}`)
	submit(t, srv, `{"id":"b","ms":30000}`)
	_, out := submit(t, srv, `{"id":"c","ms":40000}`)
	if out["rank"] != float64(3) {
		t.Errorf("동률 2명 뒤 제출 rank=%v, want 3(competition)", out["rank"])
	}
	scores := leaderboard(t, srv, "")
	wantRanks := []float64{1, 1, 3}
	for i, s := range scores {
		if r := s.(map[string]any)["rank"]; r != wantRanks[i] {
			t.Errorf("리더보드 %d행 rank=%v, want %v", i, r, wantRanks[i])
		}
	}
}

// TestSubmitValidation 은 요청 검증 실패 케이스를 테이블로 확인한다.
func TestSubmitValidation(t *testing.T) {
	srv := newServer(t)
	cases := []struct {
		name, body string
	}{
		{"빈 id", `{"id":"","ms":1000}`},
		{"공백 id", `{"id":"   ","ms":1000}`},
		{"id 33자 초과", fmt.Sprintf(`{"id":%q,"ms":1000}`, strings.Repeat("a", 33))},
		{"ms 0", `{"id":"x","ms":0}`},
		{"ms 음수", `{"id":"x","ms":-5}`},
		{"ms 24h 초과", `{"id":"x","ms":86400001}`},
		{"JSON 깨짐", `{"id":`},
		{"바디 1KB 초과", fmt.Sprintf(`{"id":%q,"ms":1000}`, strings.Repeat("a", 2048))}, // MaxBytesReader 상한
	}
	for _, c := range cases {
		code, out := submit(t, srv, c.body)
		if code != http.StatusBadRequest {
			t.Errorf("[%s] code=%d, want 400", c.name, code)
		}
		if _, ok := out["error"]; !ok {
			t.Errorf("[%s] error 필드 없음: %v", c.name, out)
		}
	}
	// 경계 안쪽은 통과해야 한다 — 32자 id, 유니코드(rune 기준 32자 이하).
	if code, _ := submit(t, srv, fmt.Sprintf(`{"id":%q,"ms":1}`, strings.Repeat("a", 32))); code != http.StatusOK {
		t.Errorf("32자 id 거부됨: %d", code)
	}
	if code, _ := submit(t, srv, `{"id":"한글닉네임","ms":1}`); code != http.StatusOK {
		t.Errorf("멀티바이트 id 거부됨: %d", code)
	}
}

// TestLeaderboardLimit 은 limit 파라미터의 기본값/클램프를 확인한다.
func TestLeaderboardLimit(t *testing.T) {
	srv := newServer(t)
	for i := 0; i < 60; i++ {
		submit(t, srv, fmt.Sprintf(`{"id":"p%d","ms":%d}`, i, 1000+i))
	}
	if n := len(leaderboard(t, srv, "")); n != 50 {
		t.Errorf("기본 limit=%d, want 50", n)
	}
	if n := len(leaderboard(t, srv, "?limit=3")); n != 3 {
		t.Errorf("limit=3 인데 %d명", n)
	}
	// 범위 밖(0/999/쓰레기)은 기본값 50 으로.
	for _, q := range []string{"?limit=0", "?limit=999", "?limit=abc"} {
		if n := len(leaderboard(t, srv, q)); n != 50 {
			t.Errorf("%s → %d명, want 50(기본값 폴백)", q, n)
		}
	}
}

// TestConcurrentSubmits 는 동시 제출이 전부 성공하는지 확인한다 —
// SetMaxOpenConns(1) 이 없으면 커넥션 풀 경합이 SQLITE_BUSY(500)로 새어
// 나온다(리뷰에서 40 동시 요청 중 38건 실패 실측). 속도 제한이 없다는 설계
// 결정 때문에 동시성은 오히려 흔한 경로다.
func TestConcurrentSubmits(t *testing.T) {
	srv := newServer(t)
	const n = 40
	errs := make(chan string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"id":"p%d","ms":%d}`, i, 1000+i)
			resp, err := http.Post(srv.URL+"/api/arena/score", "application/json", bytes.NewBufferString(body))
			if err != nil {
				errs <- err.Error()
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errs <- fmt.Sprintf("p%d → %d", i, resp.StatusCode)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Errorf("동시 제출 실패: %s", e)
	}
	if got := len(leaderboard(t, srv, "?limit=200")); got != n {
		t.Errorf("리더보드 %d명, want %d", got, n)
	}
}

// TestCORSAndMethods 는 CORS 헤더(모든 응답), OPTIONS 프리플라이트,
// 메서드 불일치 405 를 확인한다.
func TestCORSAndMethods(t *testing.T) {
	srv := newServer(t)

	resp, err := http.Get(srv.URL + "/api/arena/leaderboard")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS 헤더=%q, want *", got)
	}

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/arena/score", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("OPTIONS → %d, want 204", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/api/arena/score") // GET on POST-only
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET score → %d, want 405", resp.StatusCode)
	}

	postResp, err := http.Post(srv.URL+"/api/arena/leaderboard", "application/json", nil) // POST on GET-only
	if err != nil {
		t.Fatal(err)
	}
	postResp.Body.Close()
	if postResp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST leaderboard → %d, want 405", postResp.StatusCode)
	}
}

// leaderboardRaw 는 리더보드 응답 전체(map)를 돌려준다 — scores 배열 밖의
// 컨텍스트 필드(total/me)를 검증하는 용도.
func leaderboardRaw(t *testing.T, srv *httptest.Server, query string) map[string]any {
	t.Helper()
	resp, err := http.Get(srv.URL + "/api/arena/leaderboard" + query)
	if err != nil {
		t.Fatalf("GET leaderboard: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("leaderboard status=%d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

// TestSubmitRankContext 는 제출 응답의 경쟁 컨텍스트를 확인한다 — 전체
// 참가자 수(total), 그리고 1위가 아니면 바로 위 기록까지의 격차
// (next_id/next_gap_ms). 1위에게는 next_* 가 없어야 한다.
func TestSubmitRankContext(t *testing.T) {
	srv := newServer(t)

	code, out := submit(t, srv, `{"id":"leader","ms":40000}`)
	if code != http.StatusOK {
		t.Fatalf("1차 제출 code=%d", code)
	}
	if out["total"] != float64(1) {
		t.Errorf("1인 제출 후 total=%v, want 1", out["total"])
	}
	if _, has := out["next_id"]; has {
		t.Errorf("1위 응답에 next_id 가 있음: %v", out)
	}

	_, out = submit(t, srv, `{"id":"chaser","ms":41500}`)
	if out["rank"] != float64(2) || out["total"] != float64(2) {
		t.Fatalf("2위 컨텍스트: rank=%v total=%v, want 2/2", out["rank"], out["total"])
	}
	if out["next_id"] != "leader" {
		t.Errorf("next_id=%v, want leader", out["next_id"])
	}
	if out["next_gap_ms"] != float64(1500) {
		t.Errorf("next_gap_ms=%v, want 1500", out["next_gap_ms"])
	}

	// 격차는 재신고가 아니라 서버에 남은 best 기준 — 더 느린 재제출에도
	// next_gap_ms 가 기존 best(41500) 기준으로 유지돼야 한다.
	_, out = submit(t, srv, `{"id":"chaser","ms":90000}`)
	if out["next_gap_ms"] != float64(1500) {
		t.Errorf("나쁜 재제출 후 next_gap_ms=%v, want 1500(best 기준)", out["next_gap_ms"])
	}
}

// TestLeaderboardMeContext 는 ?me= 조회를 확인한다 — 상위 limit 밖 참가자의
// 자기 행 포함, 없는 id 는 생략, total 은 항상 포함.
func TestLeaderboardMeContext(t *testing.T) {
	srv := newServer(t)
	for i := 0; i < 5; i++ {
		submit(t, srv, fmt.Sprintf(`{"id":"p%d","ms":%d}`, i, 40000+i*1000))
	}

	out := leaderboardRaw(t, srv, "?limit=3&me=p4")
	if out["total"] != float64(5) {
		t.Errorf("total=%v, want 5", out["total"])
	}
	if len(out["scores"].([]any)) != 3 {
		t.Fatalf("scores 길이=%d, want 3", len(out["scores"].([]any)))
	}
	me, ok := out["me"].(map[string]any)
	if !ok {
		t.Fatalf("me 행이 없음: %v", out)
	}
	if me["rank"] != float64(5) || me["id"] != "p4" || me["ms"] != float64(44000) {
		t.Errorf("me=%v, want rank5/p4/44000", me)
	}

	if out := leaderboardRaw(t, srv, "?me=ghost"); out["me"] != nil {
		t.Errorf("없는 id 의 me 가 생략되지 않음: %v", out["me"])
	}
}
