package marketaux

import "context"

// EntitiesService covers the entity metadata endpoints: entity search,
// entity type list and industry list. Available on all plans.
type EntitiesService struct {
	c *Client
}

// Search looks up entities supported by Marketaux by name, symbol and
// filters. The page limit is fixed at 50 by the API.
//
// GET /entity/search
// https://www.marketaux.com/documentation#entity-search
func (s *EntitiesService) Search(ctx context.Context, p EntitySearchParams) (*EntitiesResponse, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	var out EntitiesResponse
	if err := s.c.get(ctx, "/entity/search", p.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Types returns all entity types supported by the API. The list changes
// rarely and is a good candidate for caching.
//
// GET /entity/type/list
// https://www.marketaux.com/documentation#entity-types
func (s *EntitiesService) Types(ctx context.Context) ([]EntityType, error) {
	var out StringListResponse
	if err := s.c.get(ctx, "/entity/type/list", nil, &out); err != nil {
		return nil, err
	}
	types := make([]EntityType, len(out.Data))
	for i, t := range out.Data {
		types[i] = EntityType(t)
	}
	return types, nil
}

// Industries returns all industries supported by the API.
//
// GET /entity/industry/list
// https://www.marketaux.com/documentation#entity-industries
func (s *EntitiesService) Industries(ctx context.Context) ([]string, error) {
	var out StringListResponse
	if err := s.c.get(ctx, "/entity/industry/list", nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}
