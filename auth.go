package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

func auth(db *sql.DB, w http.ResponseWriter, r *http.Request) (int, bool) {
	apiKey := r.URL.Query().Get("key")
	row := db.QueryRow("SELECT id FROM users WHERE api_key = $1", apiKey)
	var userID int
	if err := row.Scan(&userID); err == sql.ErrNoRows {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "invalid api key: %q\n", apiKey)
		return 0, false
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "db error: %s\n", err)
		return 0, false
	}
	return userID, true
}
