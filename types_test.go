package marketaux_test

import (
	"encoding/json"
	"testing"
	"time"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

func TestArticlePublishedAtFormats(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want time.Time
	}{
		{"microseconds", `"2024-11-08T01:24:00.000000Z"`, time.Date(2024, 11, 8, 1, 24, 0, 0, time.UTC)},
		{"milliseconds", `"2024-11-08T01:24:00.123Z"`, time.Date(2024, 11, 8, 1, 24, 0, 123000000, time.UTC)},
		{"seconds", `"2024-11-08T01:24:00Z"`, time.Date(2024, 11, 8, 1, 24, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var a marketaux.Article
			if err := json.Unmarshal([]byte(`{"published_at":`+tc.in+`}`), &a); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !a.PublishedAt.Equal(tc.want) {
				t.Fatalf("PublishedAt = %v, want %v", a.PublishedAt, tc.want)
			}
		})
	}
}

func TestHighlightText(t *testing.T) {
	t.Parallel()
	h := marketaux.Highlight{
		Highlight: "<em>Bitcoin</em>  (<em>BTC</em>/<em>USD</em>) &amp; more",
	}
	want := "Bitcoin (BTC/USD) & more"
	if h.Text() != want {
		t.Fatalf("Text() = %q, want %q", h.Text(), want)
	}
	if !(marketaux.Highlight{Sentiment: 0.5}.IsPositive()) {
		t.Fatal("IsPositive")
	}
	if !(marketaux.Highlight{Sentiment: -0.5}.IsNegative()) {
		t.Fatal("IsNegative")
	}
}

func TestTrendingEntityEmbedding(t *testing.T) {
	t.Parallel()
	var e marketaux.TrendingEntity
	in := `{"key":"NVDA","total_documents":22,"sentiment_avg":0.43,"score":3.79}`
	if err := json.Unmarshal([]byte(in), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Key != "NVDA" || e.TotalDocuments != 22 || e.SentimentAvg == nil || e.Score != 3.79 {
		t.Fatalf("embedded: %+v", e)
	}
}

func TestEnumValidity(t *testing.T) {
	t.Parallel()
	for _, e := range []marketaux.EntityType{
		marketaux.EntityTypeEquity, marketaux.EntityTypeIndex, marketaux.EntityTypeETF,
		marketaux.EntityTypeMutualFund, marketaux.EntityTypeCurrency, marketaux.EntityTypeCryptocurrency,
	} {
		if !e.Valid() {
			t.Errorf("entity type %q invalid", e)
		}
	}
	if marketaux.EntityType("bogus").Valid() {
		t.Error("bogus entity type")
	}
	if marketaux.Interval("bogus").Valid() || marketaux.Interval("bogus").MaxDays() != 0 {
		t.Error("bogus interval")
	}
	want := map[marketaux.Interval]int{
		marketaux.IntervalMinute: 7, marketaux.IntervalHour: 31, marketaux.IntervalDay: 1095,
		marketaux.IntervalWeek: 1825, marketaux.IntervalMonth: 1825,
		marketaux.IntervalQuarter: 3650, marketaux.IntervalYear: 3650,
	}
	for i, days := range want {
		if i.MaxDays() != days {
			t.Errorf("%s MaxDays = %d, want %d", i, i.MaxDays(), days)
		}
	}
	for _, g := range []marketaux.GroupBy{marketaux.GroupBySymbol, marketaux.GroupByExchange, marketaux.GroupByIndustry, marketaux.GroupByCountry} {
		if !g.Valid() {
			t.Errorf("group_by %q invalid", g)
		}
	}
	if marketaux.GroupBy("x").Valid() {
		t.Error("bogus group_by")
	}
	for _, s := range []marketaux.NewsSort{marketaux.NewsSortPublishedOn, marketaux.NewsSortEntityMatchScore, marketaux.NewsSortEntitySentimentScore, marketaux.NewsSortRelevanceScore} {
		if !s.Valid() {
			t.Errorf("news sort %q invalid", s)
		}
	}
	if marketaux.NewsSort("x").Valid() {
		t.Error("bogus news sort")
	}
	for _, s := range []marketaux.StatsSort{marketaux.StatsSortTotalDocuments, marketaux.StatsSortSentimentAvg} {
		if !s.Valid() {
			t.Errorf("stats sort %q invalid", s)
		}
	}
	if marketaux.StatsSort("x").Valid() {
		t.Error("bogus stats sort")
	}
	if !marketaux.SortOrderAsc.Valid() || !marketaux.SortOrderDesc.Valid() || marketaux.SortOrder("x").Valid() {
		t.Error("sort order validity")
	}
	if !marketaux.DateOrderAsc.Valid() || !marketaux.DateOrderDesc.Valid() || marketaux.DateOrder("x").Valid() {
		t.Error("date order validity")
	}
	if !marketaux.HighlightedInTitle.Valid() || !marketaux.HighlightedInMainText.Valid() || marketaux.HighlightedIn("x").Valid() {
		t.Error("highlighted_in validity")
	}
	// String methods
	if marketaux.EntityTypeEquity.String() != "equity" ||
		marketaux.IntervalDay.String() != "day" ||
		marketaux.GroupBySymbol.String() != "symbol" ||
		marketaux.NewsSortPublishedOn.String() != "published_on" ||
		marketaux.StatsSortTotalDocuments.String() != "total_documents" ||
		marketaux.SortOrderAsc.String() != "asc" ||
		marketaux.DateOrderDesc.String() != "desc" ||
		marketaux.HighlightedInTitle.String() != "title" {
		t.Error("String() methods")
	}
}
