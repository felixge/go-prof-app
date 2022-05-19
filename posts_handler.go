package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

/*
#cgo LDFLAGS: -lgmp
#include <stdlib.h>

#include <gmp.h>

long *side_effect;

// cpuHog tries to waste about 1ms of CPU time.
long cpuHog() {
	// call plain malloc, which gives us an invocation statically built in
	// to the program
	long *p = malloc(1000*sizeof(long));
	side_effect = p;
	free(p);

	// Do some GMP stuff to get memory allocation from a third-party shared
	// library. We use p to make sure all the operations in this function
	// are intertwined and hopefully don't get optimized away
	mpz_t foo;
	mpz_init_set_ui(foo, (unsigned long int) p);
	mpz_pow_ui(foo, foo, 42);

	long sum = mpz_odd_p(foo);
	for (long i = 0; i < 1000000; i++) {
		sum += i % 10;
	}
	mpz_clear(foo);
	return sum;
}
*/
import "C"

type PostsHandler struct {
	DB          *sql.DB
	CPUDuration time.Duration
	SQLDuration time.Duration
	// CGO determines if cgo is used for simulating the CPUDuration.
	CGO bool
}

func (h *PostsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth(h.DB, w, r)
	if !ok {
		return
	}

	posts, err := h.ioWork(r.Context(), userID)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, "ioWork: %s", err)
		return
	}

	data, err := h.cpuWork(posts)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, "cpuWork: %s", err)
		return
	}
	w.Write(data)
}

func (h *PostsHandler) ioWork(ctx context.Context, userID int) ([]*Post, error) {
	q := `SELECT id, user_id, title, body FROM posts, pg_sleep($1) WHERE user_id = $2`
	var rows *sql.Rows
	rows, err := h.DB.QueryContext(ctx, q, h.SQLDuration.Seconds(), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		var p Post
		if err = rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Body); err != nil {
			return nil, err
		}
		posts = append(posts, &p)
	}
	return posts, nil
}

func (h *PostsHandler) cpuWork(posts []*Post) ([]byte, error) {
	var (
		wg   sync.WaitGroup
		stop = make(chan struct{})
		data []byte
	)
	wg.Add(1)
	if h.CGO {
		go cgoCPUHog(posts, &data, &wg, stop)
	} else {
		go goCPUHog(posts, &data, &wg, stop)
	}
	time.Sleep(h.CPUDuration)
	close(stop)
	wg.Wait()
	return data, nil
}

//go:noinline
func cgoCPUHog(posts []*Post, data *[]byte, wg *sync.WaitGroup, stop chan struct{}) {
	defer wg.Done()

	// Call malloc through C.malloc because this is a special case that has
	// to be profiled without going through any Go functions
	p := C.malloc(4096)
	defer C.free(p)
loop:
	for {
		select {
		case <-stop:
			break loop
		default:
			C.cpuHog()
		}
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(posts); err != nil {
		return
	}
	*data = buf.Bytes()
}

//go:noinline
func goCPUHog(posts []*Post, data *[]byte, wg *sync.WaitGroup, stop chan struct{}) {
	defer wg.Done()

	for {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(posts); err != nil {
			return
		}
		select {
		case <-stop:
			*data = buf.Bytes()
			return
		default:
			buf.Reset()
		}
	}
}

type Post struct {
	ID     int
	UserID int
	Title  string
	Body   string
}
