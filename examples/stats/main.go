// Command stats demonstrates the stats endpoints. They require a Standard
// plan or higher.
//
// Usage:
//
//	MARKETAUX_API_TOKEN=... go run ./examples/stats
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tigusigalpa/marketaux-go"
)

func main() {
	token := os.Getenv("MARKETAUX_API_TOKEN")
	if token == "" {
		log.Fatal("set MARKETAUX_API_TOKEN")
	}

	client, err := marketaux.NewClient(token)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	series, err := client.Stats().Intraday(ctx, marketaux.StatsParams{
		Symbols:  []string{"TSLA", "AMZN", "MSFT"},
		Interval: marketaux.IntervalDay,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(series.Data), "time series points")

	agg, err := client.Stats().Aggregation(ctx, marketaux.AggregationParams{
		GroupBy: marketaux.GroupBySymbol,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range agg.Data {
		fmt.Printf("%s: %d docs\n", s.Key, s.TotalDocuments)
	}

	trending, err := client.Stats().Trending(ctx, marketaux.TrendingParams{
		Countries: []string{"us"},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range trending.Data {
		fmt.Printf("%s: score=%.2f\n", s.Key, s.Score)
	}
}
