package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type PostsHandler struct{ DB *sql.DB }

func (h PostsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth(h.DB, w, r)
	if !ok {
		return
	}

	var (
		err   error
		posts []Post
	)
	func() {
		span, ctx := tracer.StartSpanFromContext(r.Context(), "get posts")
		defer func() { span.Finish(tracer.WithError(err)) }()

		q := `SELECT id, user_id, title, body FROM posts WHERE user_id = $1`
		var rows *sql.Rows
		rows, err = h.DB.QueryContext(ctx, q, userID)
		if err != nil {
			return
		}
		defer rows.Close()

		for rows.Next() {
			var p Post
			if err = rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Body); err != nil {
				return
			}
			posts = append(posts, p)
		}
	}()

	if err != nil {
		respondErr(w, http.StatusInternalServerError, "select err: %s", err)
		return
	}

	cpuDurationS := "100ms"
	if v := r.URL.Query().Get("cpu"); v != "" {
		cpuDurationS = v
	}
	var cpuDuration time.Duration
	if cpuDuration, err = time.ParseDuration(cpuDurationS); err != nil {
		respondErr(w, http.StatusInternalServerError, "bad cpu time", err)
		return
	}

	cpuDuration = time.Duration(float64(cpuDuration) * rand.Float64())

	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		span, _ := tracer.StartSpanFromContext(r.Context(), "cpu hog")
		defer func() { span.Finish(tracer.WithError(err)) }()

		for {
			var buf bytes.Buffer
			if err = json.NewEncoder(&buf).Encode(posts); err != nil {
				return
			}
			select {
			case <-stopCh:
				io.Copy(w, &buf)
				return
			default:
			}
		}
	}()

	time.Sleep(cpuDuration)
	close(stopCh)
	wg.Wait()

	if err != nil {
		respondErr(w, http.StatusInternalServerError, "insert err: %s", err)
		return
	}
}

type Post struct {
	ID     int
	UserID int
	Title  string
	Body   string
}
