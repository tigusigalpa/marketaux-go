package marketaux

import (
	"context"
	"iter"
)

// Paginate returns an iterator that lazily walks the news feed and yields
// articles one by one. It starts at p.Page (or 1), uses the server's actual
// page size, and stops before the 20000-result bound, on error, or on break.
// An error is yielded once, with a zero Article, then iteration stops.
func (s *NewsService) Paginate(ctx context.Context, p NewsParams) iter.Seq2[Article, error] {
	return items(newsPages(ctx, s, p, 0))
}

// AllPages collects up to maxPages pages, starting at p.Page (or 1).
// maxPages <= 0 defaults to 400 pages; the 20000-result bound still applies.
// If a request fails, it returns nil and the error, discarding partial data.
// Use Paginate to process results as they arrive without storing every page.
func (s *NewsService) AllPages(ctx context.Context, p NewsParams, maxPages int) ([]Article, error) {
	if maxPages <= 0 {
		maxPages = 400
	}
	var out []Article
	for batch, err := range newsPages(ctx, s, p, maxPages) {
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
	}
	return out, nil
}

func newsPages(ctx context.Context, s *NewsService, p NewsParams, maxPages int) iter.Seq2[[]Article, error] {
	return pages(ctx, p.Page, p.Limit, maxPages, func(page, limit int) ([]Article, Meta, error) {
		q := p
		q.Page, q.Limit = page, limit
		resp, err := s.All(ctx, q)
		if err != nil {
			return nil, Meta{}, err
		}
		return resp.Data, resp.Meta, nil
	})
}

// Paginate lazily walks entity search results, starting at p.Page (or 1).
// It stops before the 20000-result bound. The API fixes the page size at 50.
func (s *EntitiesService) Paginate(ctx context.Context, p EntitySearchParams) iter.Seq2[EntityInfo, error] {
	return items(pages(ctx, p.Page, 50, 0, func(page, _ int) ([]EntityInfo, Meta, error) {
		q := p
		q.Page = page
		resp, err := s.Search(ctx, q)
		if err != nil {
			return nil, Meta{}, err
		}
		return resp.Data, resp.Meta, nil
	}))
}

// Paginate lazily walks news sources, starting at p.Page (or 1).
// It stops before the 20000-result bound. The API fixes the page size at 50.
func (s *SourcesService) Paginate(ctx context.Context, p SourcesParams) iter.Seq2[Source, error] {
	return items(pages(ctx, p.Page, 50, 0, func(page, _ int) ([]Source, Meta, error) {
		q := p
		q.Page = page
		resp, err := s.All(ctx, q)
		if err != nil {
			return nil, Meta{}, err
		}
		return resp.Data, resp.Meta, nil
	}))
}

// pages keeps pagination decisions identical across iterators and AllPages.
func pages[T any](ctx context.Context, startPage, requestedLimit, maxPages int,
	fetch func(int, int) ([]T, Meta, error)) iter.Seq2[[]T, error] {
	return func(yield func([]T, error) bool) {
		page, limit := startPage, requestedLimit
		if page == 0 {
			page = 1
		}
		for fetched := 0; maxPages <= 0 || fetched < maxPages; fetched++ {
			if err := ctx.Err(); err != nil {
				yield(nil, err)
				return
			}
			data, meta, err := fetch(page, limit)
			if err != nil {
				yield(nil, err)
				return
			}
			if len(data) == 0 || !yield(data, nil) {
				return
			}
			if meta.Limit > 0 {
				limit = meta.Limit
			}
			if limit <= 0 {
				limit = max(meta.Returned, len(data))
			}
			if meta.Returned < limit || page >= maxResultSet/limit {
				return
			}
			page++
		}
	}
}

func items[T any](batches iter.Seq2[[]T, error]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for batch, err := range batches {
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			for _, item := range batch {
				if !yield(item, nil) {
					return
				}
			}
		}
	}
}
