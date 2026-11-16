package marketaux

// EntityType is the type of a financial entity tracked by Marketaux.
// The list may grow over time; an unknown value does not break decoding.
type EntityType string

// Supported entity types.
const (
	EntityTypeEquity         EntityType = "equity"
	EntityTypeIndex          EntityType = "index"
	EntityTypeETF            EntityType = "etf"
	EntityTypeMutualFund     EntityType = "mutualfund"
	EntityTypeCurrency       EntityType = "currency"
	EntityTypeCryptocurrency EntityType = "cryptocurrency"
)

// Valid reports whether the entity type is one of the documented values.
func (e EntityType) Valid() bool {
	switch e {
	case EntityTypeEquity, EntityTypeIndex, EntityTypeETF,
		EntityTypeMutualFund, EntityTypeCurrency, EntityTypeCryptocurrency:
		return true
	}
	return false
}

// String returns the string representation of the entity type.
func (e EntityType) String() string { return string(e) }

// Interval is the bucket size of an entity stats time series.
type Interval string

// Supported stats intervals.
const (
	IntervalMinute  Interval = "minute"
	IntervalHour    Interval = "hour"
	IntervalDay     Interval = "day"
	IntervalWeek    Interval = "week"
	IntervalMonth   Interval = "month"
	IntervalQuarter Interval = "quarter"
	IntervalYear    Interval = "year"
)

// Valid reports whether the interval is one of the documented values.
func (i Interval) Valid() bool {
	switch i {
	case IntervalMinute, IntervalHour, IntervalDay, IntervalWeek,
		IntervalMonth, IntervalQuarter, IntervalYear:
		return true
	}
	return false
}

// MaxDays returns the maximum look-back period, in days, allowed by the API
// for this interval. Requesting a longer period is silently truncated by the
// server.
func (i Interval) MaxDays() int {
	switch i {
	case IntervalMinute:
		return 7
	case IntervalHour:
		return 31
	case IntervalDay:
		return 1095
	case IntervalWeek, IntervalMonth:
		return 1825
	case IntervalQuarter, IntervalYear:
		return 3650
	}
	return 0
}

// String returns the string representation of the interval.
func (i Interval) String() string { return string(i) }

// GroupBy is the aggregation dimension for stats endpoints.
type GroupBy string

// Supported group-by dimensions.
const (
	GroupBySymbol   GroupBy = "symbol"
	GroupByExchange GroupBy = "exchange"
	GroupByIndustry GroupBy = "industry"
	GroupByCountry  GroupBy = "country"
)

// Valid reports whether the group-by value is one of the documented values.
func (g GroupBy) Valid() bool {
	switch g {
	case GroupBySymbol, GroupByExchange, GroupByIndustry, GroupByCountry:
		return true
	}
	return false
}

// String returns the string representation of the group-by value.
func (g GroupBy) String() string { return string(g) }

// NewsSort is the sort field for the news feed.
type NewsSort string

// Supported news sort fields. The API documents the default as
// "published_on" even though the response field is named "published_at".
const (
	NewsSortPublishedOn          NewsSort = "published_on"
	NewsSortEntityMatchScore     NewsSort = "entity_match_score"
	NewsSortEntitySentimentScore NewsSort = "entity_sentiment_score"
	NewsSortRelevanceScore       NewsSort = "relevance_score"
)

// Valid reports whether the news sort value is one of the documented values.
func (s NewsSort) Valid() bool {
	switch s {
	case NewsSortPublishedOn, NewsSortEntityMatchScore,
		NewsSortEntitySentimentScore, NewsSortRelevanceScore:
		return true
	}
	return false
}

// String returns the string representation of the news sort value.
func (s NewsSort) String() string { return string(s) }

// StatsSort is the sort field for stats endpoints.
type StatsSort string

// Supported stats sort fields.
const (
	StatsSortTotalDocuments StatsSort = "total_documents"
	StatsSortSentimentAvg   StatsSort = "sentiment_avg"
)

// Valid reports whether the stats sort value is one of the documented values.
func (s StatsSort) Valid() bool {
	switch s {
	case StatsSortTotalDocuments, StatsSortSentimentAvg:
		return true
	}
	return false
}

// String returns the string representation of the stats sort value.
func (s StatsSort) String() string { return string(s) }

// SortOrder is the sort direction for news and stats endpoints.
type SortOrder string

// Supported sort directions.
const (
	SortOrderDesc SortOrder = "desc"
	SortOrderAsc  SortOrder = "asc"
)

// Valid reports whether the sort order is one of the documented values.
func (s SortOrder) Valid() bool {
	switch s {
	case SortOrderDesc, SortOrderAsc:
		return true
	}
	return false
}

// String returns the string representation of the sort order.
func (s SortOrder) String() string { return string(s) }

// DateOrder is the ordering of dates inside a stats time series.
type DateOrder string

// Supported date orders.
const (
	DateOrderDesc DateOrder = "desc"
	DateOrderAsc  DateOrder = "asc"
)

// Valid reports whether the date order is one of the documented values.
func (d DateOrder) Valid() bool {
	switch d {
	case DateOrderDesc, DateOrderAsc:
		return true
	}
	return false
}

// String returns the string representation of the date order.
func (d DateOrder) String() string { return string(d) }

// HighlightedIn identifies the part of an article where an entity highlight
// was found.
type HighlightedIn string

// Supported highlight locations.
const (
	HighlightedInTitle    HighlightedIn = "title"
	HighlightedInMainText HighlightedIn = "main_text"
)

// Valid reports whether the highlight location is one of the documented values.
func (h HighlightedIn) Valid() bool {
	switch h {
	case HighlightedInTitle, HighlightedInMainText:
		return true
	}
	return false
}

// String returns the string representation of the highlight location.
func (h HighlightedIn) String() string { return string(h) }
