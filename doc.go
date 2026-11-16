// Package marketaux provides an idiomatic, zero-dependency Go client for the
// Marketaux financial news API.
//
// The client covers all documented endpoints: news feeds, similar news, news
// by UUID, entity stats (time series and aggregation), trending entities,
// entity search, entity types, industries and news sources.
//
// Create a client with NewClient:
//
//	client, err := marketaux.NewClient(os.Getenv("MARKETAUX_API_TOKEN"))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	page, err := client.News().All(ctx, marketaux.NewsParams{
//		Symbols: []string{"TSLA", "AMZN"},
//	})
//
// See https://www.marketaux.com/documentation for the API reference.
package marketaux
