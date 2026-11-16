package marketaux

import "context"

// StatsService covers the entity statistics endpoints: intraday time series,
// single-period aggregations and trending entities. These endpoints require
// a Standard plan or higher; on lower plans the API responds with
// 403 endpoint_access_restricted.
type StatsService struct {
	c *Client
}

// Intraday returns a time series of entity statistics bucketed by the
// requested interval. When Symbols is empty the server returns the best
// entities by the sort metric. Dates with zero documents are included.
//
// GET /entity/stats/intraday
// https://www.marketaux.com/documentation#entity-stats
func (s *StatsService) Intraday(ctx context.Context, p StatsParams) (*TimeSeriesResponse, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	var out TimeSeriesResponse
	if err := s.c.get(ctx, "/entity/stats/intraday", p.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Aggregation returns entity statistics aggregated over a single period.
// The response Meta only carries Returned and Limit.
//
// GET /entity/stats/aggregation
// https://www.marketaux.com/documentation#entity-aggregation
func (s *StatsService) Aggregation(ctx context.Context, p AggregationParams) (*StatsResponse, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	var out StatsResponse
	if err := s.c.get(ctx, "/entity/stats/aggregation", p.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Trending returns trending entities for the period, ranked by their trending
// score. The response Meta only carries Returned and Limit.
//
// GET /entity/trending/aggregation
// https://www.marketaux.com/documentation#trending-aggregation
func (s *StatsService) Trending(ctx context.Context, p TrendingParams) (*TrendingResponse, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	var out TrendingResponse
	if err := s.c.get(ctx, "/entity/trending/aggregation", p.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}
