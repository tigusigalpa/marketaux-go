// Command entities demonstrates entity search and metadata endpoints.
//
// Usage:
//
//	MARKETAUX_API_TOKEN=... go run ./examples/entities
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

	found, err := client.Entities().Search(ctx, marketaux.EntitySearchParams{
		Search: "tesla",
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range found.Data {
		fmt.Printf("%s: %s (%s)\n", e.Symbol, e.Name, e.Type)
	}

	types, err := client.Entities().Types(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("types:", types)

	industries, err := client.Entities().Industries(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d industries, first: %s\n", len(industries), industries[0])
}
