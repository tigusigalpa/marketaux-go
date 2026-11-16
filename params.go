package marketaux

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// maxResultSet is the API-wide bound on page*limit.
const maxResultSet = 20000

// Bool returns a pointer to v. Useful for *bool parameter fields.
func Bool(v bool) *bool { return &v }

// Int returns a pointer to v.
func Int(v int) *int { return &v }

// Float64 returns a pointer to v. Useful for *float64 parameter fields.
func Float64(v float64) *float64 { return &v }

// Time returns a pointer to t.
func Time(t time.Time) *time.Time { return &t }

// Date returns a pointer to midnight UTC of the given calendar day.
func Date(year int, month time.Month, day int) *time.Time {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &t
}

// NewsParams are the query parameters of GET /news/all.
type NewsParams struct {
	Symbols          []string
	EntityTypes      []EntityType
	Industries       []string
	Countries        []string
	SentimentGTE     *float64
	SentimentLTE     *float64
	MinMatchScore    *float64
	FilterEntities   *bool
	MustHaveEntities *bool
	GroupSimilar     *bool
	Search           string
	Domains          []string
	ExcludeDomains   []string
	SourceIDs        []string
	ExcludeSourceIDs []string
	Languages        []string
	PublishedBefore  *time.Time
	PublishedAfter   *time.Time
	PublishedOn      *time.Time
	Sort             NewsSort
	SortOrder        SortOrder
	Limit            int
	Page             int
}

// SimilarParams are the query parameters of GET /news/similar/{uuid}. Unlike
// NewsParams it has no MinMatchScore, Search, Sort or SortOrder: the API does
// not accept them on this endpoint.
type SimilarParams struct {
	Symbols          []string
	EntityTypes      []EntityType
	Industries       []string
	Countries        []string
	SentimentGTE     *float64
	SentimentLTE     *float64
	FilterEntities   *bool
	MustHaveEntities *bool
	GroupSimilar     *bool
	Domains          []string
	ExcludeDomains   []string
	SourceIDs        []string
	ExcludeSourceIDs []string
	Languages        []string
	PublishedBefore  *time.Time
	PublishedAfter   *time.Time
	PublishedOn      *time.Time
	Limit            int
	Page             int
}

// StatsParams are the query parameters of GET /entity/stats/intraday.
// Stats endpoints require a Standard plan or higher.
type StatsParams struct {
	Interval         Interval
	GroupBy          GroupBy
	MinDocCount      int
	Symbols          []string
	EntityTypes      []EntityType
	Industries       []string
	Countries        []string
	SentimentAvgGTE  *float64
	SentimentAvgLTE  *float64
	SentimentGTE     *float64
	SentimentLTE     *float64
	Search           string
	Domains          []string
	ExcludeDomains   []string
	SourceIDs        []string
	ExcludeSourceIDs []string
	Languages        []string
	PublishedBefore  *time.Time
	PublishedAfter   *time.Time
	PublishedOn      *time.Time
	Sort             StatsSort
	SortOrder        SortOrder
	DateOrder        DateOrder
	Limit            int
}

// AggregationParams are the query parameters of GET /entity/stats/aggregation.
// It is StatsParams without Interval and DateOrder.
type AggregationParams struct {
	GroupBy          GroupBy
	MinDocCount      int
	Symbols          []string
	EntityTypes      []EntityType
	Industries       []string
	Countries        []string
	SentimentAvgGTE  *float64
	SentimentAvgLTE  *float64
	SentimentGTE     *float64
	SentimentLTE     *float64
	Search           string
	Domains          []string
	ExcludeDomains   []string
	SourceIDs        []string
	ExcludeSourceIDs []string
	Languages        []string
	PublishedBefore  *time.Time
	PublishedAfter   *time.Time
	PublishedOn      *time.Time
	Sort             StatsSort
	SortOrder        SortOrder
	Limit            int
}

// TrendingParams are the query parameters of GET /entity/trending/aggregation.
type TrendingParams struct {
	GroupBy          GroupBy
	MinDocCount      int
	Symbols          []string
	EntityTypes      []EntityType
	Industries       []string
	Countries        []string
	SentimentAvgGTE  *float64
	SentimentAvgLTE  *float64
	SentimentGTE     *float64
	SentimentLTE     *float64
	Search           string
	Domains          []string
	ExcludeDomains   []string
	SourceIDs        []string
	ExcludeSourceIDs []string
	Languages        []string
	PublishedBefore  *time.Time
	PublishedAfter   *time.Time
	PublishedOn      *time.Time
	Sort             StatsSort
	SortOrder        SortOrder
	Limit            int
}

// EntitySearchParams are the query parameters of GET /entity/search. The page
// limit is fixed at 50 by the API and cannot be changed.
type EntitySearchParams struct {
	Search     string
	Symbols    []string
	Exchanges  []string
	Types      []EntityType
	Industries []string
	Countries  []string
	Page       int
}

// SourcesParams are the query parameters of GET /news/sources. The page limit
// is fixed at 50 by the API and cannot be changed.
type SourcesParams struct {
	DistinctDomain *bool
	Languages      []string
	Page           int
}

// ---------------------------------------------------------------------------
// validation
// ---------------------------------------------------------------------------

type ve map[string]string

func (v ve) add(field, reason string) { v[field] = reason }

func (v ve) err() error {
	if len(v) == 0 {
		return nil
	}
	return &ValidationError{Fields: map[string]string(v)}
}

func (v ve) sentiment(field string, val *float64) {
	if val != nil && (*val < -1 || *val > 1) {
		v.add(field, "must be between -1 and 1")
	}
}

func (v ve) nonnegative(field string, val int) {
	if val < 0 {
		v.add(field, "must be >= 0")
	}
}

func (v ve) entityTypes(field string, types []EntityType) {
	for _, t := range types {
		if !t.Valid() {
			v.add(field, fmt.Sprintf("unknown entity type %q", t))
			return
		}
	}
}

func (v ve) publishedWindow(before, after, on *time.Time) {
	if on != nil && (before != nil || after != nil) {
		v.add("published_on", "cannot be combined with published_before or published_after")
		return
	}
	if before != nil && after != nil && !after.Before(*before) {
		v.add("published_after", "must be before published_before")
	}
}

func (v ve) resultSet(page, limit int) {
	if page < 0 {
		v.add("page", "must be >= 0")
	}
	if limit < 0 {
		v.add("limit", "must be >= 0")
	}
	if page > 0 && limit > 0 && int64(page)*int64(limit) > maxResultSet {
		v.add("page", "page*limit must not exceed 20000")
	}
}

// validate checks NewsParams for client-side errors.
func (p NewsParams) validate() error {
	v := ve{}
	v.sentiment("sentiment_gte", p.SentimentGTE)
	v.sentiment("sentiment_lte", p.SentimentLTE)
	v.entityTypes("entity_types", p.EntityTypes)
	v.publishedWindow(p.PublishedBefore, p.PublishedAfter, p.PublishedOn)
	v.resultSet(p.Page, p.Limit)
	if p.Sort != "" && !p.Sort.Valid() {
		v.add("sort", fmt.Sprintf("unknown news sort %q", p.Sort))
	}
	if p.SortOrder != "" {
		if !p.SortOrder.Valid() {
			v.add("sort_order", fmt.Sprintf("unknown sort order %q", p.SortOrder))
		} else if p.Sort != NewsSortEntityMatchScore && p.Sort != NewsSortEntitySentimentScore {
			v.add("sort_order", "only allowed with sort=entity_match_score or sort=entity_sentiment_score")
		}
	}
	if p.Sort == NewsSortRelevanceScore && strings.TrimSpace(p.Search) == "" {
		v.add("sort", "relevance_score requires search")
	}
	return v.err()
}

// values encodes NewsParams as URL query parameters.
func (p NewsParams) values() url.Values {
	q := url.Values{}
	setCSV(q, "symbols", p.Symbols, false)
	setTypesCSV(q, "entity_types", p.EntityTypes)
	setCSV(q, "industries", p.Industries, false)
	setCSV(q, "countries", p.Countries, true)
	setFloat(q, "sentiment_gte", p.SentimentGTE)
	setFloat(q, "sentiment_lte", p.SentimentLTE)
	setFloat(q, "min_match_score", p.MinMatchScore)
	setBool(q, "filter_entities", p.FilterEntities)
	setBool(q, "must_have_entities", p.MustHaveEntities)
	setBool(q, "group_similar", p.GroupSimilar)
	setString(q, "search", p.Search)
	setDomainsCSV(q, "domains", p.Domains)
	setDomainsCSV(q, "exclude_domains", p.ExcludeDomains)
	setCSV(q, "source_ids", p.SourceIDs, false)
	setCSV(q, "exclude_source_ids", p.ExcludeSourceIDs, false)
	setCSV(q, "language", p.Languages, true)
	setDateTime(q, "published_before", p.PublishedBefore)
	setDateTime(q, "published_after", p.PublishedAfter)
	setDate(q, "published_on", p.PublishedOn)
	if p.Sort != "" {
		q.Set("sort", string(p.Sort))
	}
	if p.SortOrder != "" {
		q.Set("sort_order", string(p.SortOrder))
	}
	setInt(q, "limit", p.Limit)
	setInt(q, "page", p.Page)
	return q
}

// validate checks SimilarParams for client-side errors.
func (p SimilarParams) validate() error {
	v := ve{}
	v.sentiment("sentiment_gte", p.SentimentGTE)
	v.sentiment("sentiment_lte", p.SentimentLTE)
	v.entityTypes("entity_types", p.EntityTypes)
	v.publishedWindow(p.PublishedBefore, p.PublishedAfter, p.PublishedOn)
	v.resultSet(p.Page, p.Limit)
	return v.err()
}

// values encodes SimilarParams as URL query parameters.
func (p SimilarParams) values() url.Values {
	q := url.Values{}
	setCSV(q, "symbols", p.Symbols, false)
	setTypesCSV(q, "entity_types", p.EntityTypes)
	setCSV(q, "industries", p.Industries, false)
	setCSV(q, "countries", p.Countries, true)
	setFloat(q, "sentiment_gte", p.SentimentGTE)
	setFloat(q, "sentiment_lte", p.SentimentLTE)
	setBool(q, "filter_entities", p.FilterEntities)
	setBool(q, "must_have_entities", p.MustHaveEntities)
	setBool(q, "group_similar", p.GroupSimilar)
	setDomainsCSV(q, "domains", p.Domains)
	setDomainsCSV(q, "exclude_domains", p.ExcludeDomains)
	setCSV(q, "source_ids", p.SourceIDs, false)
	setCSV(q, "exclude_source_ids", p.ExcludeSourceIDs, false)
	setCSV(q, "language", p.Languages, true)
	setDateTime(q, "published_before", p.PublishedBefore)
	setDateTime(q, "published_after", p.PublishedAfter)
	setDate(q, "published_on", p.PublishedOn)
	setInt(q, "limit", p.Limit)
	setInt(q, "page", p.Page)
	return q
}

// validate checks StatsParams for client-side errors.
func (p StatsParams) validate() error {
	v := ve{}
	if p.Interval != "" && !p.Interval.Valid() {
		v.add("interval", fmt.Sprintf("unknown interval %q", p.Interval))
	}
	v.statsCommon(p.GroupBy, p.MinDocCount, p.EntityTypes,
		p.SentimentAvgGTE, p.SentimentAvgLTE, p.SentimentGTE, p.SentimentLTE,
		p.PublishedBefore, p.PublishedAfter, p.PublishedOn,
		p.Sort, p.SortOrder, p.Limit)
	if p.DateOrder != "" && !p.DateOrder.Valid() {
		v.add("date_order", fmt.Sprintf("unknown date order %q", p.DateOrder))
	}
	return v.err()
}

// values encodes StatsParams as URL query parameters.
func (p StatsParams) values() url.Values {
	q := url.Values{}
	if p.Interval != "" {
		q.Set("interval", string(p.Interval))
	}
	statsValues(q, p.GroupBy, p.MinDocCount, p.Symbols, p.EntityTypes,
		p.Industries, p.Countries,
		p.SentimentAvgGTE, p.SentimentAvgLTE, p.SentimentGTE, p.SentimentLTE,
		p.Search, p.Domains, p.ExcludeDomains, p.SourceIDs, p.ExcludeSourceIDs,
		p.Languages, p.PublishedBefore, p.PublishedAfter, p.PublishedOn,
		p.Sort, p.SortOrder, p.Limit)
	if p.DateOrder != "" {
		q.Set("date_order", string(p.DateOrder))
	}
	return q
}

// validate checks AggregationParams for client-side errors.
func (p AggregationParams) validate() error {
	v := ve{}
	v.statsCommon(p.GroupBy, p.MinDocCount, p.EntityTypes,
		p.SentimentAvgGTE, p.SentimentAvgLTE, p.SentimentGTE, p.SentimentLTE,
		p.PublishedBefore, p.PublishedAfter, p.PublishedOn,
		p.Sort, p.SortOrder, p.Limit)
	return v.err()
}

// values encodes AggregationParams as URL query parameters.
func (p AggregationParams) values() url.Values {
	q := url.Values{}
	statsValues(q, p.GroupBy, p.MinDocCount, p.Symbols, p.EntityTypes,
		p.Industries, p.Countries,
		p.SentimentAvgGTE, p.SentimentAvgLTE, p.SentimentGTE, p.SentimentLTE,
		p.Search, p.Domains, p.ExcludeDomains, p.SourceIDs, p.ExcludeSourceIDs,
		p.Languages, p.PublishedBefore, p.PublishedAfter, p.PublishedOn,
		p.Sort, p.SortOrder, p.Limit)
	return q
}

// validate checks TrendingParams for client-side errors.
func (p TrendingParams) validate() error {
	v := ve{}
	v.statsCommon(p.GroupBy, p.MinDocCount, p.EntityTypes,
		p.SentimentAvgGTE, p.SentimentAvgLTE, p.SentimentGTE, p.SentimentLTE,
		p.PublishedBefore, p.PublishedAfter, p.PublishedOn,
		p.Sort, p.SortOrder, p.Limit)
	return v.err()
}

// values encodes TrendingParams as URL query parameters.
func (p TrendingParams) values() url.Values {
	q := url.Values{}
	statsValues(q, p.GroupBy, p.MinDocCount, p.Symbols, p.EntityTypes,
		p.Industries, p.Countries,
		p.SentimentAvgGTE, p.SentimentAvgLTE, p.SentimentGTE, p.SentimentLTE,
		p.Search, p.Domains, p.ExcludeDomains, p.SourceIDs, p.ExcludeSourceIDs,
		p.Languages, p.PublishedBefore, p.PublishedAfter, p.PublishedOn,
		p.Sort, p.SortOrder, p.Limit)
	return q
}

// statsCommon validates the parameter subset shared by all stats endpoints.
func (v ve) statsCommon(groupBy GroupBy, minDocCount int, types []EntityType,
	sAvgGTE, sAvgLTE, sGTE, sLTE *float64,
	before, after, on *time.Time,
	sort StatsSort, sortOrder SortOrder, limit int) {
	if groupBy != "" && !groupBy.Valid() {
		v.add("group_by", fmt.Sprintf("unknown group_by %q", groupBy))
	}
	v.nonnegative("min_doc_count", minDocCount)
	v.entityTypes("entity_types", types)
	v.sentiment("sentiment_avg_gte", sAvgGTE)
	v.sentiment("sentiment_avg_lte", sAvgLTE)
	v.sentiment("sentiment_gte", sGTE)
	v.sentiment("sentiment_lte", sLTE)
	v.publishedWindow(before, after, on)
	v.nonnegative("limit", limit)
	if sort != "" && !sort.Valid() {
		v.add("sort", fmt.Sprintf("unknown stats sort %q", sort))
	}
	if sortOrder != "" && !sortOrder.Valid() {
		v.add("sort_order", fmt.Sprintf("unknown sort order %q", sortOrder))
	}
}

// statsValues encodes the parameter subset shared by all stats endpoints.
func statsValues(q url.Values, groupBy GroupBy, minDocCount int,
	symbols []string, types []EntityType, industries, countries []string,
	sAvgGTE, sAvgLTE, sGTE, sLTE *float64,
	search string, domains, excludeDomains, sourceIDs, excludeSourceIDs, languages []string,
	before, after, on *time.Time,
	sort StatsSort, sortOrder SortOrder, limit int) {
	if groupBy != "" {
		q.Set("group_by", string(groupBy))
	}
	setInt(q, "min_doc_count", minDocCount)
	setCSV(q, "symbols", symbols, false)
	setTypesCSV(q, "entity_types", types)
	setCSV(q, "industries", industries, false)
	setCSV(q, "countries", countries, true)
	setFloat(q, "sentiment_avg_gte", sAvgGTE)
	setFloat(q, "sentiment_avg_lte", sAvgLTE)
	setFloat(q, "sentiment_gte", sGTE)
	setFloat(q, "sentiment_lte", sLTE)
	setString(q, "search", search)
	setDomainsCSV(q, "domains", domains)
	setDomainsCSV(q, "exclude_domains", excludeDomains)
	setCSV(q, "source_ids", sourceIDs, false)
	setCSV(q, "exclude_source_ids", excludeSourceIDs, false)
	setCSV(q, "language", languages, true)
	setDateTime(q, "published_before", before)
	setDateTime(q, "published_after", after)
	setDate(q, "published_on", on)
	if sort != "" {
		q.Set("sort", string(sort))
	}
	if sortOrder != "" {
		q.Set("sort_order", string(sortOrder))
	}
	setInt(q, "limit", limit)
}

// validate checks EntitySearchParams for client-side errors.
func (p EntitySearchParams) validate() error {
	v := ve{}
	v.entityTypes("types", p.Types)
	if p.Page < 0 {
		v.add("page", "must be >= 0")
	}
	return v.err()
}

// values encodes EntitySearchParams as URL query parameters.
func (p EntitySearchParams) values() url.Values {
	q := url.Values{}
	setString(q, "search", p.Search)
	setCSV(q, "symbols", p.Symbols, false)
	setCSV(q, "exchanges", p.Exchanges, false)
	setTypesCSV(q, "types", p.Types)
	setCSV(q, "industries", p.Industries, false)
	setCSV(q, "countries", p.Countries, true)
	setInt(q, "page", p.Page)
	return q
}

// validate checks SourcesParams for client-side errors.
func (p SourcesParams) validate() error {
	v := ve{}
	if p.Page < 0 {
		v.add("page", "must be >= 0")
	}
	return v.err()
}

// values encodes SourcesParams as URL query parameters.
func (p SourcesParams) values() url.Values {
	q := url.Values{}
	setBool(q, "distinct_domain", p.DistinctDomain)
	setCSV(q, "language", p.Languages, true)
	setInt(q, "page", p.Page)
	return q
}

// ---------------------------------------------------------------------------
// encoding helpers
// ---------------------------------------------------------------------------

// cleanList trims items, drops empties and deduplicates preserving order.
func cleanList(vals []string, lower bool) []string {
	if len(vals) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(vals))
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if lower {
			v = strings.ToLower(v)
		}
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// normalizeDomain strips scheme, path, query and fragment and lowercases the
// host part of a domain string.
func normalizeDomain(d string) string {
	d = strings.TrimSpace(d)
	if i := strings.Index(d, "://"); i >= 0 {
		d = d[i+3:]
	}
	if i := strings.IndexAny(d, "/?#"); i >= 0 {
		d = d[:i]
	}
	return strings.ToLower(d)
}

func setCSV(q url.Values, key string, vals []string, lower bool) {
	list := cleanList(vals, lower)
	if len(list) > 0 {
		q.Set(key, strings.Join(list, ","))
	}
}

func setTypesCSV(q url.Values, key string, types []EntityType) {
	if len(types) == 0 {
		return
	}
	vals := make([]string, len(types))
	for i, t := range types {
		vals[i] = string(t)
	}
	setCSV(q, key, vals, true)
}

func setDomainsCSV(q url.Values, key string, domains []string) {
	if len(domains) == 0 {
		return
	}
	norm := make([]string, len(domains))
	for i, d := range domains {
		norm[i] = normalizeDomain(d)
	}
	setCSV(q, key, norm, true)
}

func setBool(q url.Values, key string, v *bool) {
	if v != nil {
		q.Set(key, strconv.FormatBool(*v))
	}
}

func setFloat(q url.Values, key string, v *float64) {
	if v != nil {
		q.Set(key, strconv.FormatFloat(*v, 'f', -1, 64))
	}
}

func setInt(q url.Values, key string, v int) {
	if v > 0 {
		q.Set(key, strconv.Itoa(v))
	}
}

func setString(q url.Values, key, v string) {
	if s := strings.TrimSpace(v); s != "" {
		q.Set(key, s)
	}
}

// setDateTime encodes a date-time parameter in UTC. The API accepts several
// granularities; the SDK always sends the fullest one, Y-m-d\TH:i:s.
func setDateTime(q url.Values, key string, t *time.Time) {
	if t != nil {
		q.Set(key, t.UTC().Format("2006-01-02T15:04:05"))
	}
}

// setDate encodes a date-only parameter (Y-m-d).
func setDate(q url.Values, key string, t *time.Time) {
	if t != nil {
		q.Set(key, t.UTC().Format("2006-01-02"))
	}
}
