// Command smoke performs a live smoke test against the real API. It is not
// part of CI and requires a real token.
//
// Usage:
//
//	MARKETAUX_API_TOKEN=... go run ./examples/smoke
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tigusigalpa/marketaux-go"
)

func main() {
	token := os.Getenv("MARKETAUX_API_TOKEN")
	if token == "" {
		log.Fatal("set MARKETAUX_API_TOKEN")
	}

	var opts []marketaux.Option
	if base := os.Getenv("MARKETAUX_BASE_URL"); base != "" {
		opts = append(opts, marketaux.WithBaseURL(base))
	}
	client, err := marketaux.NewClient(token, opts...)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	types, err := client.Entities().Types(ctx)
	if err != nil {
		cancel()
		log.Fatal(err)
	}
	fmt.Println("types:", types)

	page, err := client.News().All(ctx, marketaux.NewsParams{Limit: 3})
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("found=%d returned=%d\n", page.Meta.Found, page.Meta.Returned)
	for _, a := range page.Data {
		fmt.Println("-", a.Title)
	}
}
