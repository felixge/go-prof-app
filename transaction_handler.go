package main

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"net/http"
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

	data := r.URL.Query().Get("data")
	if !doPoW(data, h.PowDifficultiy) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "pow failure")
		return
	}

	var txID int
	q := `INSERT INTO transactions (user_id, data) VALUES ($1, $2) RETURNING id`
	row := h.DB.QueryRow(q, userID, data)
	if err := row.Scan(&txID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "insert err: %s", err)
		return
	}

	fmt.Fprintf(w, "recorded transaction: %d\n", txID)
}

func doPoW(data string, difficulty int) bool {
	pow, nonce := calculatePoW(difficulty, data)
	return verifyPoW(data, pow, difficulty, nonce)
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
