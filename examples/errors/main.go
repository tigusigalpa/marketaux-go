// Command errors demonstrates typed error handling with errors.Is/As.
//
// Usage:
//
//	MARKETAUX_API_TOKEN=... go run ./examples/errors
package main

import (
	"context"
	"errors"
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

	client, err := marketaux.NewClient(token)
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.News().All(context.Background(), marketaux.NewsParams{})
	if err == nil {
		fmt.Println("request succeeded")
		return
	}

	var apiErr *marketaux.APIError
	switch {
	case errors.Is(err, marketaux.ErrRateLimit):
		if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
			fmt.Println("rate limited; retry after", apiErr.RetryAfter)
			time.Sleep(apiErr.RetryAfter)
		}
	case errors.Is(err, marketaux.ErrUsageLimit):
		fmt.Println("daily quota exhausted; limit =", apiErr.UsageLimitLimit)
	case errors.Is(err, marketaux.ErrForbidden):
		fmt.Println("endpoint not available on your plan")
	case errors.Is(err, marketaux.ErrUnauthorized):
		fmt.Println("check MARKETAUX_API_TOKEN")
	case errors.Is(err, marketaux.ErrValidation):
		var verr *marketaux.ValidationError
		if errors.As(err, &verr) {
			for field, reason := range verr.Fields {
				fmt.Printf("invalid %s: %s\n", field, reason)
			}
		}
	default:
		log.Fatal(err)
	}
}
