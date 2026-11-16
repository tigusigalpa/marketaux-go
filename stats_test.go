package marketaux_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

func TestStatsIntradayDecode(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/entity/stats/intraday" {
			t.Errorf("bad path %s", r.URL.Path)
		}
		w.Write(fixture(t, "stats_intraday.json"))
	})
	c := newTestClient(t, srv)
	resp, err := c.Stats().Intraday(context.Background(), marketaux.StatsParams{
		Interval: marketaux.IntervalDay,
		GroupBy:  marketaux.GroupBySymbol,
	})
	if err != nil {
		t.Fatalf("Intraday: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("points = %d", len(resp.Data))
	}
	want := time.Date(2024, 11, 10, 0, 0, 0, 0, time.UTC)
	if !resp.Data[0].Date.Equal(want) {
		t.Fatalf("date = %v", resp.Data[0].Date)
	}
	if len(resp.Data[0].Data) != 3 {
		t.Fatalf("buckets = %d", len(resp.Data[0].Data))
	}
	if *resp.Data[0].Data[0].SentimentAvg < 0.38 {
		t.Fatal("sentiment_avg not decoded")
	}
	// Zero-document day must keep null sentiment_avg.
	if resp.Data[1].Data[0].SentimentAvg != nil {
		t.Fatal("sentiment_avg should be nil")
	}
	if resp.Data[1].Data[0].TotalDocuments != 0 {
		t.Fatal("total_documents should be 0")
	}
}

func TestStatsAggregationDecode(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(fixture(t, "stats_aggregation.json"))
	})
	c := newTestClient(t, srv)
	resp, err := c.Stats().Aggregation(context.Background(), marketaux.AggregationParams{
		GroupBy: marketaux.GroupBySymbol,
	})
	if err != nil {
		t.Fatalf("Aggregation: %v", err)
	}
	if resp.Meta.Returned != 3 || resp.Meta.Limit != 100 {
		t.Fatalf("meta: %+v", resp.Meta)
	}
	if resp.Meta.Found != 0 || resp.Meta.Page != 0 {
		t.Fatal("aggregation meta must not have found/page")
	}
	if len(resp.Data) != 3 || resp.Data[0].Key != "TSLA" {
		t.Fatalf("data: %+v", resp.Data)
	}
}

func TestStatsTrendingDecode(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(fixture(t, "trending.json"))
	})
	c := newTestClient(t, srv)
	resp, err := c.Stats().Trending(context.Background(), marketaux.TrendingParams{
		Countries: []string{"us"},
	})
	if err != nil {
		t.Fatalf("Trending: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("data len = %d", len(resp.Data))
	}
	first := resp.Data[0]
	if first.Key != "NVDA" || first.TotalDocuments != 22 {
		t.Fatalf("trending: %+v", first)
	}
	if first.Score != 3.7920454395721324 {
		t.Fatalf("score = %v", first.Score)
	}
	if first.SentimentAvg == nil || *first.SentimentAvg < 0.4 {
		t.Fatal("embedded EntityStat not filled")
	}
}
