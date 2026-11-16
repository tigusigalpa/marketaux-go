package marketaux

import (
	"context"
	"iter"
)

// Paginate returns an iterator that lazily walks every page of the news feed
// and yields articles one by one. Iteration stops at the end of the result
// set, at the 20000-result API bound, on error, or when the consumer stops
// ranging. Errors are delivered as the second yield value; after an error the
// iterator terminates.
//
//	for article, err := range client.News().Paginate(ctx, params) {
//		if err != nil {
//			return err
//		}
//		fmt.Println(article.Title)
//	}
func (s *NewsService) Paginate(ctx context.Context, p NewsParams) iter.Seq2[Article, error] {
	return func(yield func(Article, error) bool) {
		page := p.Page
		if page <= 0 {
			page = 1
		}
		limit := p.Limit
		for {
			if err := ctx.Err(); err != nil {
				yield(Article{}, err)
				return
			}
			q := p
			q.Page = page
			resp, err := s.All(ctx, q)
			if err != nil {
				yield(Article{}, err)
				return
			}
			for _, a := range resp.Data {
				if !yield(a, nil) {
					return
				}
			}
			if limit <= 0 {
				limit = resp.Meta.Limit
			}
			effective := limit
			if effective <= 0 {
				effective = resp.Meta.Returned
			}
			if len(resp.Data) == 0 || resp.Meta.Returned < effective ||
				int64(page)*int64(effective) >= maxResultSet {
				return
			}
			page++
		}
	}
}

// AllPages collects up to maxPages pages of news into a slice. It is a
// convenience wrapper around All with a safeguard against unbounded result
// sets. maxPages <= 0 means 400 pages, which cannot exceed the API bound of
// 20000 results at the default page sizes.
func (s *NewsService) AllPages(ctx context.Context, p NewsParams, maxPages int) ([]Article, error) {
	if maxPages <= 0 {
		maxPages = 400
	}
	page := p.Page
	if page <= 0 {
		page = 1
	}
	limit := p.Limit
	var out []Article
	for i := 0; i < maxPages; i++ {
		q := p
		q.Page = page
		resp, err := s.All(ctx, q)
		if err != nil {
			return nil, err
		}
		out = append(out, resp.Data...)
		if limit <= 0 {
			limit = resp.Meta.Limit
		}
		effective := limit
		if effective <= 0 {
			effective = resp.Meta.Returned
		}
		if len(resp.Data) == 0 || resp.Meta.Returned < effective ||
			int64(page)*int64(effective) >= maxResultSet {
			break
		}
		page++
	}
	return out, nil
}

// Paginate returns an iterator that lazily walks every page of entity search
// results. The API fixes the page limit at 50.
func (s *EntitiesService) Paginate(ctx context.Context, p EntitySearchParams) iter.Seq2[EntityInfo, error] {
	return func(yield func(EntityInfo, error) bool) {
		page := p.Page
		if page <= 0 {
			page = 1
		}
		for {
			if err := ctx.Err(); err != nil {
				yield(EntityInfo{}, err)
				return
			}
			q := p
			q.Page = page
			resp, err := s.Search(ctx, q)
			if err != nil {
				yield(EntityInfo{}, err)
				return
			}
			for _, e := range resp.Data {
				if !yield(e, nil) {
					return
				}
			}
			limit := resp.Meta.Limit
			if limit <= 0 {
				limit = 50
			}
			if len(resp.Data) == 0 || resp.Meta.Returned < limit {
				return
			}
			page++
		}
	}
}

// Paginate returns an iterator that lazily walks every page of news sources.
// The API fixes the page limit at 50.
func (s *SourcesService) Paginate(ctx context.Context, p SourcesParams) iter.Seq2[Source, error] {
	return func(yield func(Source, error) bool) {
		page := p.Page
		if page <= 0 {
			page = 1
		}
		for {
			if err := ctx.Err(); err != nil {
				yield(Source{}, err)
				return
			}
			q := p
			q.Page = page
			resp, err := s.All(ctx, q)
			if err != nil {
				yield(Source{}, err)
				return
			}
			for _, src := range resp.Data {
				if !yield(src, nil) {
					return
				}
			}
			limit := resp.Meta.Limit
			if limit <= 0 {
				limit = 50
			}
			if len(resp.Data) == 0 || resp.Meta.Returned < limit {
				return
			}
			page++
		}
	}
}
