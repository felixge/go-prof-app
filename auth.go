package main

import (
	"database/sql"
	"net/http"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func auth(db *sql.DB, w http.ResponseWriter, r *http.Request) (int, bool) {
	span, ctx := tracer.StartSpanFromContext(r.Context(), "auth")
	var err error
	defer func() { span.Finish(tracer.WithError(err)) }()

	apiKey := r.URL.Query().Get("key")
	row := db.QueryRowContext(ctx, "SELECT id FROM users WHERE api_key = $1", apiKey)
	var userID int
	if err = row.Scan(&userID); err == sql.ErrNoRows {
		respondErr(w, http.StatusForbidden, "invalid api key: %q\n", apiKey)
		return 0, false
	} else if err != nil {
		respondErr(w, http.StatusInternalServerError, "db error: %s\n", err)
		return 0, false
	}
	return userID, true
}
