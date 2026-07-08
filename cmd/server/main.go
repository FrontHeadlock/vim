// Arena 리더보드 서버 진입점 — 본체(internal/arena)에 위임만 하는 얇은
// 껍데기다(cmd/web/main.go 와 같은 관례).
package main

import (
	"flag"
	"log"
	"net/http"

	"vimquest/internal/arena"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "arena.db", `sqlite db path (":memory:" for ephemeral)`)
	flag.Parse()

	db, err := arena.OpenDB(*dbPath)
	if err != nil {
		log.Fatalf("arena: %v", err)
	}
	defer db.Close()

	log.Printf("arena server listening on %s (db=%s)", *addr, *dbPath)
	log.Fatal(http.ListenAndServe(*addr, arena.NewHandler(db)))
}
