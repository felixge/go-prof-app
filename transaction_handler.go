package main

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"net/http"
	"sync"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type TransactionHandler struct {
	DB             *sql.DB
	PowDifficultiy int
}

func (h TransactionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth(h.DB, w, r)
	if !ok {
		return
	}

	powSpan, _ := tracer.StartSpanFromContext(r.Context(), "pow")
	data := r.URL.Query().Get("data")
	if !doPoW(data, h.PowDifficultiy) {
		powSpan.Finish()
		respondErr(w, http.StatusInternalServerError, "pow failure")
		return
	}
	powSpan.Finish()

	span, ctx := tracer.StartSpanFromContext(r.Context(), "insert")
	var txID int
	q := `INSERT INTO transactions (user_id, data) VALUES ($1, $2) RETURNING id`
	row := h.DB.QueryRowContext(ctx, q, userID, data)
	if err := row.Scan(&txID); err != nil {
		span.Finish()
		respondErr(w, http.StatusInternalServerError, "insert err: %s", err)
		return
	}
	span.Finish()

	fmt.Fprintf(w, "recorded transaction: %d\n", txID)
}

func doPoW(data string, difficulty int) bool {
	var (
		wg     sync.WaitGroup
		result bool
	)

	// Do work in goroutine to make it more difficult to correlated in
	// profiling without endpoint filter feature.
	wg.Add(1)
	go func() {
		defer wg.Done()
		pow, nonce := calculatePoW(difficulty, data)
		result = verifyPoW(data, pow, difficulty, nonce)
	}()
	wg.Wait()

	return result
}

func calculatePoW(difficulty int, data string) (string, int) {
	for nonce := 0; ; nonce++ {
		pow := hash(data, nonce)
		if verifyPoW(data, pow, difficulty, nonce) {
			return pow, nonce
		}
	}
}

func hash(data string, nonce int) string {
	s := fmt.Sprintf("%s-%d", data, nonce)
	hash := sha1.Sum([]byte(s))
	return fmt.Sprintf("%x", string(hash[:]))
}

func verifyPoW(data, pow string, difficulty, nonce int) bool {
	got := hash(data, nonce)
	if got != pow {
		return false
	}
	return verifyDifficulty(pow, difficulty)
}

func verifyDifficulty(pow string, difficulty int) bool {
	for j := 0; j < difficulty; j++ {
		if pow[j] != '0' {
			return false
		}
	}
	return true
}
