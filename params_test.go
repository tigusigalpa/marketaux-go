package marketaux

import (
	"errors"
	"testing"
	"time"
)

func TestNewsParamsValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		p     NewsParams
		field string
	}{
		{"sentiment gte high", NewsParams{SentimentGTE: Float64(1.5)}, "sentiment_gte"},
		{"sentiment gte low", NewsParams{SentimentGTE: Float64(-1.5)}, "sentiment_gte"},
		{"sentiment lte", NewsParams{SentimentLTE: Float64(2)}, "sentiment_lte"},
		{"negative limit", NewsParams{Limit: -1}, "limit"},
		{"negative page", NewsParams{Page: -1}, "page"},
		{"result set too big", NewsParams{Page: 201, Limit: 100}, "page"},
		{"result set ok", NewsParams{Page: 200, Limit: 100}, ""},
		{"bad entity type", NewsParams{EntityTypes: []EntityType{"bogus"}}, "entity_types"},
		{"bad sort", NewsParams{Sort: "bogus"}, "sort"},
		{"bad sort order", NewsParams{SortOrder: "bogus", Sort: NewsSortEntityMatchScore}, "sort_order"},
		{"order with published_on", NewsParams{Sort: NewsSortPublishedOn, SortOrder: SortOrderAsc}, "sort_order"},
		{"order with relevance", NewsParams{Sort: NewsSortRelevanceScore, SortOrder: SortOrderAsc, Search: "x"}, "sort_order"},
		{"relevance without search", NewsParams{Sort: NewsSortRelevanceScore}, "sort"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.p.validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected *ValidationError, got %v", err)
			}
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("expected ErrValidation, got %v", err)
			}
			if _, ok := ve.Fields[tc.field]; !ok {
				t.Fatalf("expected field %q in %v", tc.field, ve.Fields)
			}
		})
	}
}

func TestNewsParamsDateValidation(t *testing.T) {
	t.Parallel()
	on := Date(2024, 1, 1)
	before := Date(2024, 2, 1)
	after := Date(2024, 1, 15)
	later := Date(2024, 3, 1)

	cases := []struct {
		name  string
		p     NewsParams
		field string
	}{
		{"on with before", NewsParams{PublishedOn: on, PublishedBefore: before}, "published_on"},
		{"on with after", NewsParams{PublishedOn: on, PublishedAfter: after}, "published_on"},
		{"after gt before", NewsParams{PublishedAfter: later, PublishedBefore: before}, "published_after"},
		{"window ok", NewsParams{PublishedAfter: after, PublishedBefore: before}, ""},
		{"on alone", NewsParams{PublishedOn: on}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.p.validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected ValidationError, got %v", err)
			}
			if _, ok := ve.Fields[tc.field]; !ok {
				t.Fatalf("expected field %q in %v", tc.field, ve.Fields)
			}
		})
	}
}

func TestNewsParamsMultipleErrors(t *testing.T) {
	t.Parallel()
	p := NewsParams{
		SentimentGTE: Float64(5),
		Limit:        -1,
		Sort:         "bogus",
	}
	var ve *ValidationError
	err := p.validate()
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(ve.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %v", ve.Fields)
	}
}

func TestSimilarParamsValues(t *testing.T) {
	t.Parallel()
	p := SimilarParams{
		Symbols:        []string{"TSLA"},
		FilterEntities: Bool(true),
		Limit:          10,
		Page:           2,
	}
	q := p.values()
	if q.Get("symbols") != "TSLA" || q.Get("filter_entities") != "true" ||
		q.Get("limit") != "10" || q.Get("page") != "2" {
		t.Fatalf("unexpected values: %v", q)
	}
	for _, banned := range []string{"search", "sort", "sort_order", "min_match_score"} {
		if q.Has(banned) {
			t.Fatalf("similar params must not send %q", banned)
		}
	}
	if err := p.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestStatsParamsValues(t *testing.T) {
	t.Parallel()
	p := StatsParams{
		Interval:        IntervalDay,
		GroupBy:         GroupBySymbol,
		MinDocCount:     5,
		Symbols:         []string{"TSLA", "MSFT"},
		SentimentAvgGTE: Float64(0.1),
		Search:          "earnings",
		Sort:            StatsSortSentimentAvg,
		SortOrder:       SortOrderDesc,
		DateOrder:       DateOrderAsc,
		Limit:           20,
	}
	q := p.values()
	want := map[string]string{
		"interval":          "day",
		"group_by":          "symbol",
		"min_doc_count":     "5",
		"symbols":           "TSLA,MSFT",
		"sentiment_avg_gte": "0.1",
		"search":            "earnings",
		"sort":              "sentiment_avg",
		"sort_order":        "desc",
		"date_order":        "asc",
		"limit":             "20",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
	if err := p.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestStatsParamsValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		p     StatsParams
		field string
	}{
		{"bad interval", StatsParams{Interval: "bogus"}, "interval"},
		{"bad group_by", StatsParams{GroupBy: "bogus"}, "group_by"},
		{"bad date_order", StatsParams{DateOrder: "bogus"}, "date_order"},
		{"neg min_doc", StatsParams{MinDocCount: -1}, "min_doc_count"},
		{"sentiment avg", StatsParams{SentimentAvgGTE: Float64(2)}, "sentiment_avg_gte"},
		{"ok", StatsParams{Interval: IntervalMonth, GroupBy: GroupByIndustry}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.p.validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected ValidationError, got %v", err)
			}
			if _, ok := ve.Fields[tc.field]; !ok {
				t.Fatalf("expected field %q in %v", tc.field, ve.Fields)
			}
		})
	}
}

func TestAggregationParamsValues(t *testing.T) {
	t.Parallel()
	p := AggregationParams{
		GroupBy:    GroupByCountry,
		Industries: []string{"Technology"},
		Limit:      50,
	}
	q := p.values()
	if q.Get("group_by") != "country" || q.Get("industries") != "Technology" {
		t.Fatalf("unexpected values: %v", q)
	}
	for _, banned := range []string{"interval", "date_order", "page"} {
		if q.Has(banned) {
			t.Fatalf("aggregation params must not send %q", banned)
		}
	}
	if err := p.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := (AggregationParams{GroupBy: "bogus"}).validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestTrendingParamsValues(t *testing.T) {
	t.Parallel()
	p := TrendingParams{
		GroupBy:   GroupByExchange,
		Countries: []string{"US"},
	}
	q := p.values()
	if q.Get("group_by") != "exchange" || q.Get("countries") != "us" {
		t.Fatalf("unexpected values: %v", q)
	}
	if q.Has("interval") || q.Has("date_order") {
		t.Fatal("trending params must not send interval/date_order")
	}
	if err := p.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := (TrendingParams{SentimentLTE: Float64(-2)}).validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestEntitySearchParams(t *testing.T) {
	t.Parallel()
	p := EntitySearchParams{
		Search:    "tesla",
		Symbols:   []string{"TSLA"},
		Exchanges: []string{"NASDAQ"},
		Types:     []EntityType{EntityTypeEquity},
		Countries: []string{"us"},
		Page:      2,
	}
	q := p.values()
	want := map[string]string{
		"search":    "tesla",
		"symbols":   "TSLA",
		"exchanges": "NASDAQ",
		"types":     "equity",
		"countries": "us",
		"page":      "2",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
	if q.Has("limit") {
		t.Fatal("entity search must not send limit")
	}
	if err := p.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := (EntitySearchParams{Page: -1}).validate(); err == nil {
		t.Fatal("expected error for negative page")
	}
	if err := (EntitySearchParams{Types: []EntityType{"bogus"}}).validate(); err == nil {
		t.Fatal("expected error for bad type")
	}
}

func TestSourcesParams(t *testing.T) {
	t.Parallel()
	p := SourcesParams{
		DistinctDomain: Bool(true),
		Languages:      []string{"EN"},
		Page:           3,
	}
	q := p.values()
	if q.Get("distinct_domain") != "true" || q.Get("language") != "en" || q.Get("page") != "3" {
		t.Fatalf("unexpected values: %v", q)
	}
	if q.Has("limit") {
		t.Fatal("sources must not send limit")
	}
	if err := p.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := (SourcesParams{Page: -1}).validate(); err == nil {
		t.Fatal("expected error for negative page")
	}
}

func TestNormalizeDomain(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want string }{
		{"adweek.com", "adweek.com"},
		{" AdWeek.COM ", "adweek.com"},
		{"https://adweek.com/path", "adweek.com"},
		{"http://adweek.com/p?q=1#f", "adweek.com"},
		{"www.example.com/x?y=2", "www.example.com"},
		{"example.com?query=1", "example.com"},
		{"example.com#frag", "example.com"},
	}
	for _, tc := range cases {
		if got := normalizeDomain(tc.in); got != tc.want {
			t.Errorf("normalizeDomain(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCleanList(t *testing.T) {
	t.Parallel()
	got := cleanList([]string{" a ", "", "b", "a", "  ", "A"}, false)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "A" {
		t.Fatalf("cleanList = %v", got)
	}
	got = cleanList([]string{" A ", "a"}, true)
	if len(got) != 1 || got[0] != "a" {
		t.Fatalf("cleanList lower = %v", got)
	}
	if got := cleanList(nil, false); got != nil {
		t.Fatalf("cleanList nil = %v", got)
	}
}

func TestHelpers(t *testing.T) {
	t.Parallel()
	if *Bool(true) != true || *Int(5) != 5 || *Float64(1.5) != 1.5 {
		t.Fatal("pointer helpers broken")
	}
	tm := time.Date(2026, time.November, 16, 1, 2, 3, 0, time.UTC)
	if !Time(tm).Equal(tm) {
		t.Fatal("Time helper broken")
	}
	d := Date(2026, time.November, 16)
	if d.Hour() != 0 || d.Location() != time.UTC {
		t.Fatal("Date must be midnight UTC")
	}
}

func FuzzNewsParamsValues(f *testing.F) {
	f.Add("TSLA", "en", "adweek.com", "x.com-1", "Technology")
	f.Add("", " ", "https://evil/?a=b", "\x00", "multi")
	f.Fuzz(func(t *testing.T, sym, lang, dom, src, ind string) {
		p := NewsParams{
			Symbols:          []string{sym},
			Languages:        []string{lang},
			Domains:          []string{dom},
			SourceIDs:        []string{src},
			Industries:       []string{ind},
			Search:           sym,
			ExcludeDomains:   []string{dom},
			ExcludeSourceIDs: []string{src},
		}
		q := p.values()
		if q == nil {
			t.Fatal("values returned nil")
		}
	})
}

func BenchmarkNewsParamsValues(b *testing.B) {
	day := Date(2024, 1, 1)
	p := NewsParams{
		Symbols:        []string{"TSLA", "AMZN", "MSFT"},
		EntityTypes:    []EntityType{EntityTypeEquity},
		Countries:      []string{"us"},
		Languages:      []string{"en"},
		SentimentGTE:   Float64(0),
		FilterEntities: Bool(true),
		PublishedOn:    day,
		Limit:          50,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.values()
	}
}
