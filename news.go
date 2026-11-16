package marketaux

import (
	"context"
	"net/url"
	"regexp"
)

// uuidRe matches canonical RFC 4122 UUIDs.
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// validateUUID returns a ValidationError when uuid is not a canonical UUID.
func validateUUID(uuid string) error {
	if !uuidRe.MatchString(uuid) {
		return &ValidationError{Fields: map[string]string{
			"uuid": "must be a valid RFC 4122 UUID",
		}}
	}
	return nil
}

// NewsService covers the /news/* endpoints: the main news feed, similar news
// lookup and fetching a single article by UUID. Available on all plans.
type NewsService struct {
	c *Client
}

// All returns the latest financial news matching the given filters.
//
// GET /news/all
// https://www.marketaux.com/documentation#finance-news
func (s *NewsService) All(ctx context.Context, p NewsParams) (*NewsResponse, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	var out NewsResponse
	if err := s.c.get(ctx, "/news/all", p.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Similar returns articles similar to the article with the given UUID.
// Unknown UUIDs may produce either an empty list or a 404 error; handle both.
//
// GET /news/similar/{uuid}
// https://www.marketaux.com/documentation#similar-news
func (s *NewsService) Similar(ctx context.Context, uuid string, p SimilarParams) (*NewsResponse, error) {
	if err := validateUUID(uuid); err != nil {
		return nil, err
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	var out NewsResponse
	path := "/news/similar/" + url.PathEscape(uuid)
	if err := s.c.get(ctx, path, p.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ByUUID returns a single article by its UUID. Missing articles produce a
// 404 error mapped to ErrNotFound.
//
// GET /news/uuid/{uuid}
// https://www.marketaux.com/documentation#news-by-uuid
func (s *NewsService) ByUUID(ctx context.Context, uuid string) (*Article, error) {
	if err := validateUUID(uuid); err != nil {
		return nil, err
	}
	var out Article
	path := "/news/uuid/" + url.PathEscape(uuid)
	if err := s.c.get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
