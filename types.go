package marketaux

import (
	"html"
	"regexp"
	"strings"
	"time"
)

// Meta is the pagination envelope returned by most list endpoints. For
// aggregation endpoints (stats aggregation and trending) only Returned and
// Limit are populated; Found and Page remain zero.
type Meta struct {
	Found    int `json:"found"`
	Returned int `json:"returned"`
	Limit    int `json:"limit"`
	Page     int `json:"page"`
}

// Article is a single news article returned by the news endpoints.
type Article struct {
	UUID           string    `json:"uuid"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Keywords       string    `json:"keywords"`
	Snippet        string    `json:"snippet"`
	URL            string    `json:"url"`
	ImageURL       string    `json:"image_url"`
	Language       string    `json:"language"`
	PublishedAt    time.Time `json:"published_at"`
	Source         string    `json:"source"`
	RelevanceScore *float64  `json:"relevance_score"`
	Entities       []Entity  `json:"entities"`
	Similar        []Article `json:"similar"`
}

// HasEntities reports whether the article has at least one recognized entity.
func (a Article) HasEntities() bool { return len(a.Entities) > 0 }

// IsAbout reports whether the article mentions the given symbol
// (case-insensitive).
func (a Article) IsAbout(symbol string) bool {
	for _, e := range a.Entities {
		if strings.EqualFold(e.Symbol, symbol) {
			return true
		}
	}
	return false
}

// Entity is a financial entity detected inside an article.
type Entity struct {
	Symbol         string      `json:"symbol"`
	Name           string      `json:"name"`
	Exchange       *string     `json:"exchange"`
	ExchangeLong   *string     `json:"exchange_long"`
	Country        string      `json:"country"`
	Type           EntityType  `json:"type"`
	Industry       string      `json:"industry"`
	MatchScore     float64     `json:"match_score"`
	SentimentScore float64     `json:"sentiment_score"`
	Highlights     []Highlight `json:"highlights"`
}

// IsPositive reports whether the entity sentiment score is above zero.
func (e Entity) IsPositive() bool { return e.SentimentScore > 0 }

// IsNegative reports whether the entity sentiment score is below zero.
func (e Entity) IsNegative() bool { return e.SentimentScore < 0 }

// TypeKnown reports whether the entity type is one of the documented values.
// Unknown values still decode; the API may introduce new types at any time.
func (e Entity) TypeKnown() bool { return e.Type.Valid() }

// Highlight is a text fragment in which an entity was detected. The Highlight
// field contains <em>...</em> markup around the entity mention.
type Highlight struct {
	Highlight     string        `json:"highlight"`
	Sentiment     float64       `json:"sentiment"`
	HighlightedIn HighlightedIn `json:"highlighted_in"`
}

var (
	emTagRe  = regexp.MustCompile(`</?em>`)
	spacesRe = regexp.MustCompile(`\s+`)
)

// Text returns the highlight with <em> markup removed, HTML entities
// unescaped and whitespace normalized.
func (h Highlight) Text() string {
	s := emTagRe.ReplaceAllString(h.Highlight, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(spacesRe.ReplaceAllString(s, " "))
}

// IsPositive reports whether the highlight sentiment is above zero.
func (h Highlight) IsPositive() bool { return h.Sentiment > 0 }

// IsNegative reports whether the highlight sentiment is below zero.
func (h Highlight) IsNegative() bool { return h.Sentiment < 0 }

// Source is a news source usable in the domains and source_ids filters.
// SourceID is nil when the response was requested with distinct_domain=true.
type Source struct {
	SourceID *string `json:"source_id"`
	Domain   string  `json:"domain"`
	Language string  `json:"language"`
}

// EntityInfo is an entity as returned by the entity search endpoint.
type EntityInfo struct {
	Symbol       string     `json:"symbol"`
	Name         string     `json:"name"`
	Type         EntityType `json:"type"`
	Industry     string     `json:"industry"`
	Exchange     *string    `json:"exchange"`
	ExchangeLong *string    `json:"exchange_long"`
	Country      string     `json:"country"`
}

// EntityStat is a single aggregated bucket of entity statistics. SentimentAvg
// is nil when there were no documents in the bucket.
type EntityStat struct {
	Key            string   `json:"key"`
	TotalDocuments int      `json:"total_documents"`
	SentimentAvg   *float64 `json:"sentiment_avg"`
}

// TrendingEntity is an aggregated entity with a trending score.
type TrendingEntity struct {
	EntityStat
	Score float64 `json:"score"`
}

// TimeSeriesPoint is one bucket of an intraday stats time series.
type TimeSeriesPoint struct {
	Date time.Time    `json:"date"`
	Data []EntityStat `json:"data"`
}

// NewsResponse is the response envelope of /news/all and /news/similar/{uuid}.
type NewsResponse struct {
	Meta Meta      `json:"meta"`
	Data []Article `json:"data"`
}

// EntitiesResponse is the response envelope of /entity/search.
type EntitiesResponse struct {
	Meta Meta         `json:"meta"`
	Data []EntityInfo `json:"data"`
}

// SourcesResponse is the response envelope of /news/sources.
type SourcesResponse struct {
	Meta Meta     `json:"meta"`
	Data []Source `json:"data"`
}

// StatsResponse is the response envelope of /entity/stats/aggregation. Its
// Meta only carries Returned and Limit.
type StatsResponse struct {
	Meta Meta         `json:"meta"`
	Data []EntityStat `json:"data"`
}

// TrendingResponse is the response envelope of /entity/trending/aggregation.
// Its Meta only carries Returned and Limit.
type TrendingResponse struct {
	Meta Meta             `json:"meta"`
	Data []TrendingEntity `json:"data"`
}

// TimeSeriesResponse is the response body of /entity/stats/intraday. It has
// no meta object; dates with zero documents are included.
type TimeSeriesResponse struct {
	Data []TimeSeriesPoint `json:"data"`
}

// StringListResponse is the response envelope of /entity/type/list and
// /entity/industry/list.
type StringListResponse struct {
	Data []string `json:"data"`
}
