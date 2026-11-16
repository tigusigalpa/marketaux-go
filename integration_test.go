//go:build integration

package marketaux_test

import (
	"context"
	"errors"
	"os"
	"testing"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

// Run with:
//
//	MARKETAUX_API_TOKEN=... go test -tags=integration ./...
//
// These tests hit the real API and are skipped unless the token is set.
func TestIntegration(t *testing.T) {
	token := os.Getenv("MARKETAUX_API_TOKEN")
	if token == "" {
		t.Skip("MARKETAUX_API_TOKEN not set")
	}
	base := os.Getenv("MARKETAUX_BASE_URL")
	var opts []marketaux.Option
	if base != "" {
		opts = append(opts, marketaux.WithBaseURL(base))
	}
	client, err := marketaux.NewClient(token, opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx := context.Background()

	t.Run("types", func(t *testing.T) {
		types, err := client.Entities().Types(ctx)
		if err != nil {
			t.Fatalf("Types: %v", err)
		}
		if len(types) == 0 {
			t.Fatal("no types")
		}
	})

	t.Run("news", func(t *testing.T) {
		resp, err := client.News().All(ctx, marketaux.NewsParams{
			Languages: []string{"en"},
			Limit:     3,
		})
		if err != nil {
			t.Fatalf("All: %v", err)
		}
		if resp.Meta.Returned == 0 {
			t.Skip("empty feed")
		}
	})

	t.Run("bad token maps to ErrUnauthorized", func(t *testing.T) {
		bad, err := marketaux.NewClient("definitely-wrong", opts...)
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		_, err = bad.Entities().Types(ctx)
		if !errors.Is(err, marketaux.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})
}
