package marketaux

import "context"

// SourcesService covers the /news/sources endpoint which lists the news
// sources usable in the domains and source_ids filters. Available on all
// plans.
type SourcesService struct {
	c *Client
}

// All returns the news sources known to Marketaux. The page limit is fixed
// at 50 by the API. With DistinctDomain the response is grouped by unique
// domain and SourceID is nil.
//
// GET /news/sources
// https://www.marketaux.com/documentation#sources
func (s *SourcesService) All(ctx context.Context, p SourcesParams) (*SourcesResponse, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	var out SourcesResponse
	if err := s.c.get(ctx, "/news/sources", p.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}
