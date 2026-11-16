package marketaux_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

// ExampleNewClient shows the minimal setup. It uses a test server so the
// example is deterministic and needs no API token.
func ExampleNewClient() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"meta":{"found":1,"returned":1,"limit":1,"page":1},"data":[{"uuid":"u","title":"Markets rally","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x.com","relevance_score":null,"entities":[],"similar":[]}]}`))
	}))
	defer srv.Close()

	client, err := marketaux.NewClient("token",
		marketaux.WithBaseURL(srv.URL),
		marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	page, err := client.News().All(context.Background(), marketaux.NewsParams{
		Symbols: []string{"TSLA"},
	})
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	for _, a := range page.Data {
		fmt.Println(a.Title)
	}
	// Output: Markets rally
}

// ExampleNewsService_Paginate shows iterating over the whole result set with
// range-over-func.
func ExampleNewsService_Paginate() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"meta":{"found":1,"returned":1,"limit":50,"page":1},"data":[{"uuid":"u","title":"One","description":"","keywords":"","snippet":"","url":"","image_url":"","language":"en","published_at":"2024-01-01T00:00:00.000Z","source":"x.com","relevance_score":null,"entities":[],"similar":[]}]}`))
	}))
	defer srv.Close()

	client, _ := marketaux.NewClient("token",
		marketaux.WithBaseURL(srv.URL),
		marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1}),
	)
	for article, err := range client.News().Paginate(context.Background(), marketaux.NewsParams{}) {
		if err != nil {
			fmt.Println("unexpected error:", err)
			return
		}
		fmt.Println(article.Title)
	}
	// Output: One
}
