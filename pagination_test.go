package marketaux_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

func collect(t *testing.T, c *marketaux.Client, p marketaux.NewsParams) (int, error) {
	t.Helper()
	n := 0
	var outErr error
	for _, err := range c.News().Paginate(context.Background(), p) {
		if err != nil {
			outErr = err
			break
		}
		n++
	}
	return n, outErr
}

func TestPaginateMultiplePages(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		page := r.URL.Query().Get("page")
		switch page {
		case "", "1":
			w.Write([]byte(`{"meta":{"found":3,"returned":2,"limit":2,"page":1},"data":[
				{"uuid":"a","title":"t1","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]},
				{"uuid":"b","title":"t2","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]}]}`))
		case "2":
			w.Write([]byte(`{"meta":{"found":3,"returned":1,"limit":2,"page":2},"data":[
				{"uuid":"c","title":"t3","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]}]}`))
		default:
			w.Write([]byte(`{"meta":{"found":3,"returned":0,"limit":2,"page":3},"data":[]}`))
		}
	})
	c := newTestClient(t, srv)
	n, err := collect(t, c, marketaux.NewsParams{Limit: 2})
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if n != 3 {
		t.Fatalf("articles = %d, want 3", n)
	}
	if calls.Load() != 2 {
		t.Fatalf("requests = %d, want 2", calls.Load())
	}
}

func TestPaginateStopsOnEmpty(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"meta":{"found":0,"returned":0,"limit":50,"page":1},"data":[]}`))
	})
	c := newTestClient(t, srv)
	n, err := collect(t, c, marketaux.NewsParams{})
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if n != 0 || calls.Load() != 1 {
		t.Fatalf("n=%d calls=%d", n, calls.Load())
	}
}

func TestPaginateStopsAt20000(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"meta":{"found":99999,"returned":100,"limit":100,"page":1},"data":[{"uuid":"a","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]}]}`))
	})
	c := newTestClient(t, srv)
	// Start at page 200 with limit 100: page*limit == 20000 → must stop.
	n, err := collect(t, c, marketaux.NewsParams{Page: 200, Limit: 100})
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("requests = %d, want 1", calls.Load())
	}
	if n != 1 {
		t.Fatalf("articles = %d", n)
	}
}

func TestPaginateUsesMetaLimit(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		page := r.URL.Query().Get("page")
		if page == "2" {
			w.Write([]byte(`{"meta":{"found":4,"returned":1,"limit":3,"page":2},"data":[{"uuid":"b","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]}]}`))
			return
		}
		w.Write([]byte(`{"meta":{"found":4,"returned":3,"limit":3,"page":1},"data":[{"uuid":"a","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]},{"uuid":"a2","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]},{"uuid":"a3","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]}]}`))
	})
	c := newTestClient(t, srv)
	// Limit unset → use Meta.Limit (3) to decide whether to continue.
	n, err := collect(t, c, marketaux.NewsParams{})
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if n != 4 || calls.Load() != 2 {
		t.Fatalf("n=%d calls=%d", n, calls.Load())
	}
}

func TestPaginateAPIError(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(fixture(t, "error_401.json"))
	})
	c := newTestClient(t, srv)
	_, err := collect(t, c, marketaux.NewsParams{})
	if !errors.Is(err, marketaux.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestPaginateContextCancel(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"meta":{"found":99,"returned":1,"limit":1,"page":1},"data":[{"uuid":"a","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]}]}`))
	})
	c := newTestClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	n := 0
	var outErr error
	for _, err := range c.News().Paginate(ctx, marketaux.NewsParams{Limit: 1}) {
		if err != nil {
			outErr = err
			break
		}
		n++
		cancel()
	}
	cancel()
	if outErr == nil {
		t.Fatal("expected error after cancel")
	}
}

func TestAllPages(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		page := r.URL.Query().Get("page")
		if page == "1" || page == "" {
			w.Write([]byte(`{"meta":{"found":4,"returned":2,"limit":2,"page":1},"data":[{"uuid":"a","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]},{"uuid":"b","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]}]}`))
			return
		}
		w.Write([]byte(`{"meta":{"found":4,"returned":2,"limit":2,"page":` + page + `},"data":[{"uuid":"c","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]},{"uuid":"d","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[],"similar":[]}]}`))
	})
	c := newTestClient(t, srv)
	arts, err := c.News().AllPages(context.Background(), marketaux.NewsParams{Limit: 2}, 2)
	if err != nil {
		t.Fatalf("AllPages: %v", err)
	}
	if len(arts) != 4 {
		t.Fatalf("articles = %d", len(arts))
	}
	if calls.Load() != 2 {
		t.Fatalf("requests = %d, want 2 (maxPages)", calls.Load())
	}
}

func TestEntitiesPaginate(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("page") == "2" {
			w.Write([]byte(`{"meta":{"found":51,"returned":1,"limit":50,"page":2},"data":[{"symbol":"B","name":"b","type":"equity","industry":"i","exchange":null,"exchange_long":null,"country":"us"}]}`))
			return
		}
		data := ""
		for i := 0; i < 50; i++ {
			if i > 0 {
				data += ","
			}
			data += `{"symbol":"A","name":"a","type":"equity","industry":"i","exchange":null,"exchange_long":null,"country":"us"}`
		}
		fmt.Fprintf(w, `{"meta":{"found":51,"returned":50,"limit":50,"page":1},"data":[%s]}`, data)
	})
	c := newTestClient(t, srv)
	n := 0
	for _, err := range c.Entities().Paginate(context.Background(), marketaux.EntitySearchParams{Search: "x"}) {
		if err != nil {
			t.Fatalf("paginate: %v", err)
		}
		n++
	}
	if n != 51 || calls.Load() != 2 {
		t.Fatalf("n=%d calls=%d", n, calls.Load())
	}
}

func TestSourcesPaginate(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("page") == "2" {
			w.Write([]byte(`{"meta":{"found":51,"returned":1,"limit":50,"page":2},"data":[{"source_id":"b-1","domain":"b.com","language":"en"}]}`))
			return
		}
		data := ""
		for i := 0; i < 50; i++ {
			if i > 0 {
				data += ","
			}
			data += fmt.Sprintf(`{"source_id":"a-%d","domain":"a.com","language":"en"}`, i)
		}
		fmt.Fprintf(w, `{"meta":{"found":51,"returned":50,"limit":50,"page":1},"data":[%s]}`, data)
	})
	c := newTestClient(t, srv)
	n := 0
	for _, err := range c.Sources().Paginate(context.Background(), marketaux.SourcesParams{}) {
		if err != nil {
			t.Fatalf("paginate: %v", err)
		}
		n++
	}
	if n != 51 || calls.Load() != 2 {
		t.Fatalf("n=%d calls=%d", n, calls.Load())
	}
}
