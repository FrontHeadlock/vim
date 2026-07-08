package arena

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxIDRunes = 32
	maxMS      = 24 * 60 * 60 * 1000 // 24시간 — 신고 시간의 상식적 상한
	defaultLim = 50
	minLim     = 1
	maxLim     = 200
)

// NewHandler 는 db 를 물린 Arena HTTP API 를 만든다. 모든 응답에 개방형
// CORS 헤더를 실어 보낸다 — 보호할 세션/쿠키가 애초에 없고, 개발 배치에선
// 프론트(python http.server)와 이 서버가 서로 다른 origin 이기 때문이다.
func NewHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/arena/score", handleScore(db))
	mux.HandleFunc("/api/arena/leaderboard", handleLeaderboard(db))
	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

type scoreReq struct {
	ID string `json:"id"`
	MS int64  `json:"ms"`
}

func handleScore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req scoreReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		id := strings.TrimSpace(req.ID)
		if id == "" {
			writeError(w, http.StatusBadRequest, "id required")
			return
		}
		if utf8.RuneCountInString(id) > maxIDRunes {
			writeError(w, http.StatusBadRequest, "id too long")
			return
		}
		if req.MS <= 0 {
			writeError(w, http.StatusBadRequest, "ms must be positive")
			return
		}
		if req.MS > maxMS {
			writeError(w, http.StatusBadRequest, "ms out of range")
			return
		}

		// upsert→best→rank 세 쿼리는 트랜잭션이 아니다 — 동시 제출이 사이에
		// 끼면 rank 가 한두 계단 어긋날 수 있다. 클라이언트 신고 시간을 그대로
		// 믿는 이 서버의 신뢰 모델에서 원자성은 살 가치가 없는 보증이라 의도적
		// 으로 두는 것이지, 원자적이라고 착각해서가 아니다.
		now := time.Now().UnixMilli()
		if err := upsertScore(db, id, req.MS, now); err != nil {
			writeError(w, http.StatusInternalServerError, "store failed")
			return
		}
		best, err := bestFor(db, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "store failed")
			return
		}
		rank, err := rankOf(db, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "store failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"best_ms": best,
			"rank":    rank,
		})
	}
}

func handleLeaderboard(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		limit := defaultLim
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		if limit < minLim || limit > maxLim {
			limit = defaultLim
		}
		scores, err := topScores(db, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "store failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"scores": scores})
	}
}
