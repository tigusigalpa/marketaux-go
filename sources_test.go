package marketaux_test

import (
	"context"
	"net/http"
	"testing"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

func TestSourcesDecode(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/news/sources" {
			t.Errorf("bad path %s", r.URL.Path)
		}
		w.Write(fixture(t, "sources.json"))
	})
	c := newTestClient(t, srv)
	resp, err := c.Sources().All(context.Background(), marketaux.SourcesParams{
		Languages: []string{"en"},
	})
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if resp.Meta.Found != 5327 || resp.Meta.Limit != 50 {
		t.Fatalf("meta: %+v", resp.Meta)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len = %d", len(resp.Data))
	}
	s := resp.Data[0]
	if s.SourceID == nil || *s.SourceID != "adweek.com-1" {
		t.Fatalf("source_id = %v", s.SourceID)
	}
	if s.Domain != "adweek.com" || s.Language != "en" {
		t.Fatalf("source: %+v", s)
	}
}

func TestSourcesDistinctDomain(t *testing.T) {
	t.Parallel()
	var distinct string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		distinct = r.URL.Query().Get("distinct_domain")
		w.Write([]byte(`{"meta":{"found":3000,"returned":1,"limit":50,"page":1},"data":[{"source_id":null,"domain":"adweek.com","language":"en"}]}`))
	})
	c := newTestClient(t, srv)
	resp, err := c.Sources().All(context.Background(), marketaux.SourcesParams{
		DistinctDomain: marketaux.Bool(true),
	})
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if distinct != "true" {
		t.Fatalf("distinct_domain = %q", distinct)
	}
	if resp.Data[0].SourceID != nil {
		t.Fatal("source_id should be nil with distinct_domain")
	}
}
