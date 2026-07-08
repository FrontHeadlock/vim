// Package arena 는 Arena 시간공격 리더보드 백엔드다 — internal/game·engine·
// store 와 완전히 분리된 관심사로, cmd/server 만 이 패키지를 import 하고
// 게임 패키지와는 어느 방향으로도 import 관계가 없다. 그 경계가 게임의
// 무네트워크·무cgo TinyGo 빌드(크기 예산)를 그대로 지킨다.
package arena

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Score 는 리더보드 한 행이다.
type Score struct {
	Rank int    `json:"rank"`
	ID   string `json:"id"`
	MS   int64  `json:"ms"`
}

const schema = `
CREATE TABLE IF NOT EXISTS arena_scores (
	player_id  TEXT PRIMARY KEY,
	best_ms    INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_arena_best ON arena_scores(best_ms ASC);
`

// OpenDB 는 path 의 SQLite DB 를 열고(없으면 생성) 스키마를 보장한다.
// path 는 ":memory:"(휘발성/테스트용)일 수 있다.
func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("arena: open db: %w", err)
	}
	// 커넥션 1개로 고정 — database/sql 의 커넥션 풀이 SQLite 단일 파일에
	// 여러 커넥션을 열면 동시 제출이 SQLITE_BUSY(→500)로 터지고(40 동시
	// 요청에서 38건 실패 실측), ":memory:" 는 커넥션마다 독립된 빈 DB 가
	// 되어 방금 쓴 점수가 리더보드 조회에 안 보인다. 리더보드 하나 규모에서
	// 직렬화 비용은 무시 가능하고, busy_timeout/WAL 튜닝보다 한 줄로 두 결함을
	// 모두 없애는 쪽을 택했다.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("arena: init schema: %w", err)
	}
	return db, nil
}

// upsertScore 는 ms 를 기록하되 그 id 의 역대 최고(최소) 시간만 유지한다.
// updated_at 은 기록이 실제로 개선될 때만 전진한다 — 그래야 리더보드
// 동률 타이브레이크(updated_at ASC)가 나쁜 재제출에 흔들리지 않는다.
func upsertScore(db *sql.DB, id string, ms, nowMillis int64) error {
	_, err := db.Exec(`
		INSERT INTO arena_scores(player_id, best_ms, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(player_id) DO UPDATE SET
			updated_at = CASE WHEN excluded.best_ms < arena_scores.best_ms
			                  THEN excluded.updated_at ELSE arena_scores.updated_at END,
			best_ms    = MIN(arena_scores.best_ms, excluded.best_ms)
	`, id, ms, nowMillis)
	return err
}

// bestFor 는 upsert 직후의 현재 best_ms 를 돌려준다.
func bestFor(db *sql.DB, id string) (int64, error) {
	var best int64
	err := db.QueryRow(`SELECT best_ms FROM arena_scores WHERE player_id = ?`, id).Scan(&best)
	return best, err
}

// rankOf 는 id 의 1-based 순위(competition ranking — 더 빠른 기록 수 +1)를
// 돌려준다. 전제: id 는 이미 존재해야 한다(upsert 직후에만 호출) — 없는 id 면
// 서브쿼리가 NULL 이 되어 비교가 전부 거짓, 1위로 잘못 답한다.
func rankOf(db *sql.DB, id string) (int, error) {
	var rank int
	err := db.QueryRow(`
		SELECT COUNT(*) + 1 FROM arena_scores
		WHERE best_ms < (SELECT best_ms FROM arena_scores WHERE player_id = ?)
	`, id).Scan(&rank)
	return rank, err
}

// topScores 는 빠른 순으로 상위 limit 개의 기록을 돌려준다.
func topScores(db *sql.DB, limit int) ([]Score, error) {
	rows, err := db.Query(`
		SELECT player_id, best_ms FROM arena_scores
		ORDER BY best_ms ASC, updated_at ASC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// competition ranking(동률은 같은 순위, 다음 순위는 건너뜀) — rankOf 의
	// COUNT(더 빠른 기록)+1 과 같은 정의다. 순차 번호를 매기면 동률일 때
	// 제출 응답의 rank 와 리더보드 표시가 서로 어긋난다.
	out := []Score{}
	prevMS := int64(-1)
	rank := 0
	for i := 1; rows.Next(); i++ {
		var s Score
		if err := rows.Scan(&s.ID, &s.MS); err != nil {
			return nil, err
		}
		if s.MS != prevMS {
			rank = i
			prevMS = s.MS
		}
		s.Rank = rank
		out = append(out, s)
	}
	return out, rows.Err()
}
