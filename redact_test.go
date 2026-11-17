package marketaux

import (
	"net/url"
	"strings"
	"testing"
)

func TestRedactURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"token masked",
			"https://api.marketaux.com/v1/news/all?api_token=secret&limit=3",
			"api_token=%2A%2A%2A&limit=3",
		},
		{
			"no token unchanged",
			"https://api.marketaux.com/v1/news/all?limit=3",
			"limit=3",
		},
		{"invalid escape", "https://example.com?api_token=secret%ZZ&limit=3", "api_token=%2A%2A%2A"},
		{"encoded key and invalid value", "https://example.com?api%5Ftoken=secret%ZZ", "api%5Ftoken=%2A%2A%2A"},
		{"semicolon in query", "https://example.com?api_token=secret;bad&limit=3", "limit=3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := RedactURL(tc.in)
			if strings.Contains(got, "secret") {
				t.Fatalf("token leaked: %q", got)
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("RedactURL = %q, want substring %q", got, tc.want)
			}
		})
	}
}

func TestRedactURLUnparseable(t *testing.T) {
	t.Parallel()
	// Even unparseable URLs must not leak the token.
	in := "http://[::1/api_token=secret"
	got := RedactURL(in)
	if strings.Contains(got, "secret") {
		t.Fatalf("token leaked: %q", got)
	}
}

func TestMaskTokenParam(t *testing.T) {
	t.Parallel()
	in := "x?api_token=abc&y=1&api_token=def#frag"
	got := maskTokenParam(in)
	if strings.Contains(got, "abc") || strings.Contains(got, "def") {
		t.Fatalf("tokens leaked: %q", got)
	}
	if !strings.Contains(got, "y=1") {
		t.Fatalf("other params lost: %q", got)
	}
}

func TestRedactValues(t *testing.T) {
	t.Parallel()
	q := url.Values{}
	q.Set("api_token", "secret")
	q.Set("limit", "3")
	got := redactValues(q)
	if strings.Contains(got, "secret") {
		t.Fatalf("token leaked: %q", got)
	}
	if !strings.Contains(got, "limit=3") {
		t.Fatalf("param lost: %q", got)
	}
	// original values untouched
	if q.Get("api_token") != "secret" {
		t.Fatal("redactValues mutated input")
	}

	q2 := url.Values{}
	q2.Set("limit", "3")
	if got := redactValues(q2); got != "limit=3" {
		t.Fatalf("redactValues = %q", got)
	}
}
