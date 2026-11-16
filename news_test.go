package marketaux_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

func TestNewsAllDecode(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(fixture(t, "news_all.json"))
	})
	c := newTestClient(t, srv)
	resp, err := c.News().All(context.Background(), marketaux.NewsParams{})
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if resp.Meta.Found != 140037 || resp.Meta.Returned != 2 || resp.Meta.Page != 1 {
		t.Fatalf("meta: %+v", resp.Meta)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("data len = %d", len(resp.Data))
	}
	a := resp.Data[0]
	if a.UUID != "70cb577e-c2dd-4dde-b501-f713823a4939" {
		t.Fatalf("uuid = %q", a.UUID)
	}
	if a.RelevanceScore != nil {
		t.Fatal("relevance_score should be nil")
	}
	if !a.HasEntities() {
		t.Fatal("expected entities")
	}
	if !a.IsAbout("tsla") {
		t.Fatal("IsAbout case-insensitive failed")
	}
	if a.IsAbout("MSFT") {
		t.Fatal("IsAbout false positive")
	}
	e := a.Entities[0]
	if e.Symbol != "TSLA" || e.Type != marketaux.EntityTypeEquity {
		t.Fatalf("entity: %+v", e)
	}
	if e.Exchange != nil || e.ExchangeLong != nil {
		t.Fatal("exchange should be nil")
	}
	if !e.TypeKnown() || !e.IsPositive() || e.IsNegative() {
		t.Fatal("entity helpers broken")
	}
	if len(e.Highlights) != 1 {
		t.Fatalf("highlights = %d", len(e.Highlights))
	}
	h := e.Highlights[0]
	if h.HighlightedIn != marketaux.HighlightedInMainText {
		t.Fatalf("highlighted_in = %q", h.HighlightedIn)
	}
	want := "., majority-owned by Trump, and Tesl[+253 characters]"
	if h.Text() != want {
		t.Fatalf("Text() = %q, want %q", h.Text(), want)
	}
	wantPub := time.Date(2024, 11, 8, 1, 24, 0, 0, time.UTC)
	if !a.PublishedAt.Equal(wantPub) {
		t.Fatalf("published_at = %v", a.PublishedAt)
	}
	// Second article: null optional fields.
	b := resp.Data[1]
	if b.Description != "" || b.Snippet != "" || b.ImageURL != "" {
		t.Fatal("null fields should decode to empty strings")
	}
	if b.HasEntities() {
		t.Fatal("expected no entities")
	}
}

func TestNewsSimilarDecode(t *testing.T) {
	t.Parallel()
	uuid := "70cb577e-c2dd-4dde-b501-f713823a4939"
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/news/similar/"+uuid {
			t.Errorf("bad path %s", r.URL.Path)
		}
		w.Write(fixture(t, "news_similar.json"))
	})
	c := newTestClient(t, srv)
	resp, err := c.News().Similar(context.Background(), uuid, marketaux.SimilarParams{})
	if err != nil {
		t.Fatalf("Similar: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len = %d", len(resp.Data))
	}
	if resp.Data[0].RelevanceScore == nil {
		t.Fatal("relevance_score expected for similar news")
	}
}

func TestNewsSimilarBadUUID(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("request should not reach server")
	})
	c := newTestClient(t, srv)
	_, err := c.News().Similar(context.Background(), "not-a-uuid", marketaux.SimilarParams{})
	if !errors.Is(err, marketaux.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestNewsByUUID(t *testing.T) {
	t.Parallel()
	uuid := "9b9fb67e-9438-4263-815e-ef6a7d36af07"
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/news/uuid/"+uuid {
			t.Errorf("bad path %s", r.URL.Path)
		}
		w.Write(fixture(t, "news_uuid.json"))
	})
	c := newTestClient(t, srv)
	a, err := c.News().ByUUID(context.Background(), uuid)
	if err != nil {
		t.Fatalf("ByUUID: %v", err)
	}
	if a.UUID != uuid {
		t.Fatalf("uuid = %q", a.UUID)
	}
	e := a.Entities[0]
	if e.Exchange == nil || *e.Exchange != "CC" {
		t.Fatal("exchange should be CC")
	}
	if e.Country != "global" {
		t.Fatalf("country = %q", e.Country)
	}
	if e.Type != marketaux.EntityTypeCryptocurrency {
		t.Fatalf("type = %q", e.Type)
	}
	if !e.IsNegative() {
		t.Fatal("expected negative sentiment")
	}
	if len(a.Similar) != 0 {
		t.Fatal("uuid response must not have similar")
	}
}

func TestNewsByUUIDNotFound(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write(fixture(t, "error_404.json"))
	})
	c := newTestClient(t, srv)
	_, err := c.News().ByUUID(context.Background(), "9b9fb67e-9438-4263-815e-ef6a7d36af07")
	if !errors.Is(err, marketaux.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if errors.Is(err, marketaux.ErrEndpointNotFound) {
		t.Fatal("resource_not_found must not map to ErrEndpointNotFound")
	}
}

func TestNewsByUUIDBadUUID(t *testing.T) {
	t.Parallel()
	c, err := marketaux.NewClient(testToken)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "xyz", "9b9fb67e94384263815eef6a7d36af07"} {
		if _, err := c.News().ByUUID(context.Background(), bad); !errors.Is(err, marketaux.ErrValidation) {
			t.Fatalf("uuid %q: expected ErrValidation, got %v", bad, err)
		}
	}
}

func TestUnknownEntityTypeDecodes(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"meta":{"found":1,"returned":1,"limit":1,"page":1},"data":[{"uuid":"1","title":"t","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x","relevance_score":null,"entities":[{"symbol":"X","name":"x","exchange":null,"exchange_long":null,"country":"us","type":"futuretype","industry":null,"match_score":1,"sentiment_score":0,"highlights":[]}],"similar":[]}]}`))
	})
	c := newTestClient(t, srv)
	resp, err := c.News().All(context.Background(), marketaux.NewsParams{})
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	e := resp.Data[0].Entities[0]
	if e.Type != marketaux.EntityType("futuretype") {
		t.Fatalf("type = %q", e.Type)
	}
	if e.TypeKnown() {
		t.Fatal("futuretype must not be known")
	}
}
