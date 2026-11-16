package marketaux_test

import (
	"context"
	"net/http"
	"testing"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

func TestEntitiesSearchDecode(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/entity/search" {
			t.Errorf("bad path %s", r.URL.Path)
		}
		if r.URL.Query().Get("search") != "tesla" {
			t.Errorf("search = %q", r.URL.Query().Get("search"))
		}
		w.Write(fixture(t, "entity_search.json"))
	})
	c := newTestClient(t, srv)
	resp, err := c.Entities().Search(context.Background(), marketaux.EntitySearchParams{
		Search: "tesla",
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Meta.Limit != 50 {
		t.Fatalf("limit = %d", resp.Meta.Limit)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len = %d", len(resp.Data))
	}
	e := resp.Data[0]
	if e.Symbol != "TSLA" || e.Type != marketaux.EntityTypeEquity || e.Country != "us" {
		t.Fatalf("entity: %+v", e)
	}
	if e.Exchange != nil {
		t.Fatal("exchange should be nil")
	}
}

func TestEntitiesTypesDecode(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/entity/type/list" {
			t.Errorf("bad path %s", r.URL.Path)
		}
		w.Write(fixture(t, "entity_types.json"))
	})
	c := newTestClient(t, srv)
	types, err := c.Entities().Types(context.Background())
	if err != nil {
		t.Fatalf("Types: %v", err)
	}
	if len(types) != 6 {
		t.Fatalf("types = %v", types)
	}
	if types[0] != marketaux.EntityTypeEquity {
		t.Fatalf("types[0] = %q", types[0])
	}
	for _, ty := range types {
		if !ty.Valid() {
			t.Fatalf("unexpected type %q", ty)
		}
	}
}

func TestEntitiesIndustriesDecode(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/entity/industry/list" {
			t.Errorf("bad path %s", r.URL.Path)
		}
		w.Write(fixture(t, "entity_industries.json"))
	})
	c := newTestClient(t, srv)
	industries, err := c.Entities().Industries(context.Background())
	if err != nil {
		t.Fatalf("Industries: %v", err)
	}
	if len(industries) != 16 {
		t.Fatalf("industries = %d", len(industries))
	}
	if industries[0] != "Technology" {
		t.Fatalf("industries[0] = %q", industries[0])
	}
}
