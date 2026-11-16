// Command news prints the latest financial news for a set of symbols.
//
// Usage:
//
//	MARKETAUX_API_TOKEN=... go run ./examples/news
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

	page, err := client.News().All(context.Background(), marketaux.NewsParams{
		Symbols:        []string{"TSLA", "AMZN"},
		Languages:      []string{"en"},
		FilterEntities: marketaux.Bool(true),
		SentimentGTE:   marketaux.Float64(0),
		Limit:          10,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, article := range page.Data {
		fmt.Println(article.Title, "-", article.URL)
		for _, entity := range article.Entities {
			fmt.Printf("  %s (%s) sentiment=%.2f\n",
				entity.Symbol, entity.Name, entity.SentimentScore)
		}
	}
}
