// Command pagination shows lazy iteration over the full news feed with
// range-over-func iterators.
//
// Usage:
//
//	MARKETAUX_API_TOKEN=... go run ./examples/pagination
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

	n := 0
	for article, err := range client.News().Paginate(context.Background(), marketaux.NewsParams{
		Symbols: []string{"TSLA"},
	}) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(article.PublishedAt.Format("2006-01-02"), "-", article.Title)
		if n++; n >= 20 {
			break // stop early; the iterator is lazy
		}
	}
}
