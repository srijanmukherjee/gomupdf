package gomupdf

// Concurrency (v1.2)
// ==================
//
// MuPDF forbids sharing an fz_context across threads, and its lazy page-tree
// parsing makes even a lock-shared document unsafe to load pages from in
// parallel. The robust model is therefore one of independent state: each worker
// gets its own context AND its own document, opened over the same immutable
// input bytes. Workers share no mutable MuPDF state, so they render and extract
// different pages truly in parallel — "fan a large PDF across all your cores."
//
// MapPages packages that into one call: it fans a set of page indices across a
// bounded worker pool where each worker owns an independent read-only backend
// wrapped in a private Document — so the full *Page read API works unchanged and
// in parallel. RenderPages and TextByPageConcurrent are thin conveniences.
//
// Scope and safety:
//   - These helpers are for READ/RENDER operations. Workers see the document's
//     content as it was opened from bytes; unsaved in-memory mutations are not
//     reflected, and mutating through a worker *Page is undefined.
//   - Only stream-opened, unencrypted documents run in parallel. In-memory
//     documents (NewPDF), encrypted documents, and backends without the
//     capability transparently fall back to serial execution.
//   - Results are returned in the same order as the input page slice. The first
//     error cancels the remaining work and is returned.

import (
	"context"
	"errors"
	"runtime"
	"sync"
)

// PoolOption configures a concurrent page operation.
type PoolOption func(*poolConfig)

type poolConfig struct {
	workers int
}

// WithWorkers sets the maximum number of concurrent workers (cloned contexts).
// Values < 1 are ignored. The pool never spawns more workers than pages. The
// default is runtime.GOMAXPROCS(0).
func WithWorkers(n int) PoolOption {
	return func(c *poolConfig) {
		if n >= 1 {
			c.workers = n
		}
	}
}

// MapPages applies fn to each page index in pages concurrently and returns the
// results in input order. Each worker goroutine receives a *Page bound to its
// own cloned, read-only view of the document, so fn may call any read or render
// method (GetText, Words, StructuredText, Search, Pixmap, …) safely in parallel.
//
// fn MUST NOT mutate the document (insert/delete pages, draw, annotate, save);
// the worker pages share the underlying content and mutation is unsupported.
//
// The document is held locked for the whole call, so it must not be Closed or
// written by another goroutine meanwhile. If ctx is canceled, in-flight work is
// allowed to finish and ctx.Err() is returned. The first fn error cancels the
// remaining pages and is returned. If the backend cannot clone (or the document
// is closed), MapPages runs fn serially instead.
func MapPages[T any](ctx context.Context, d *Document, pages []int, fn func(*Page) (T, error), opts ...PoolOption) ([]T, error) {
	if d == nil {
		return nil, errors.New("gomupdf: nil document")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := poolConfig{workers: runtime.GOMAXPROCS(0)}
	for _, o := range opts {
		o(&cfg)
	}

	d.mu.Lock()
	if d.b == nil {
		d.mu.Unlock()
		return nil, errors.New("gomupdf: document closed")
	}
	n := d.b.pageCount()
	for _, p := range pages {
		if p < 0 || p >= n {
			d.mu.Unlock()
			return nil, errors.New("gomupdf: page out of range")
		}
	}
	cb, ok := d.b.(concurrentBackend)
	// Fall back to serial when the backend lacks the capability, the document is
	// encrypted (workers reopen the buffer fresh and have no password), or a
	// probe clone fails (e.g. in-memory documents with no backing bytes). The
	// probe both decides viability and surfaces the reason cheaply, before any
	// goroutines spawn.
	var probe docBackend
	var probeErr error
	if ok && !d.encrypted {
		probe, probeErr = cb.cloneWorker()
	}
	if !ok || d.encrypted || probeErr != nil {
		// Run serially without holding the lock (the public Page methods take
		// the lock themselves).
		d.mu.Unlock()
		return mapPagesSerial(ctx, d, pages, fn)
	}
	// Hold the lock for the parallel section: workers operate on independent
	// backends, never on d.mu, so this only blocks concurrent mutation/close of
	// the base.
	defer d.mu.Unlock()
	return mapPagesConcurrent(ctx, cb, probe, pages, cfg.workers, fn)
}

func mapPagesSerial[T any](ctx context.Context, d *Document, pages []int, fn func(*Page) (T, error)) ([]T, error) {
	out := make([]T, len(pages))
	for i, pno := range pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		r, err := fn(&Page{doc: d, Number: pno})
		if err != nil {
			return nil, err
		}
		out[i] = r
	}
	return out, nil
}

func mapPagesConcurrent[T any](ctx context.Context, cb concurrentBackend, probe docBackend, pages []int, workers int, fn func(*Page) (T, error)) ([]T, error) {
	n := len(pages)
	if n == 0 {
		if probe != nil {
			probe.close()
		}
		return nil, nil
	}
	if workers < 1 {
		workers = 1
	}
	if workers > n {
		workers = n
	}

	results := make([]T, n)
	jobs := make(chan int) // positions into pages

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)
	fail := func(err error) {
		errOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		// Worker 0 reuses the already-built probe clone; the rest build their own.
		var seed docBackend
		if w == 0 {
			seed = probe
		}
		go func(seed docBackend) {
			defer wg.Done()
			clone := seed
			if clone == nil {
				c, err := cb.cloneWorker()
				if err != nil {
					fail(err)
					return
				}
				clone = c
			}
			defer clone.close()
			// Private Document wrapper: its mutex is uncontended (one worker),
			// and it exposes the full read API over the cloned backend.
			wd := &Document{b: clone}
			for pos := range jobs {
				if cctx.Err() != nil {
					return
				}
				r, err := fn(&Page{doc: wd, Number: pages[pos]})
				if err != nil {
					fail(err)
					return
				}
				results[pos] = r
			}
		}(seed)
	}

	for pos := 0; pos < n; pos++ {
		select {
		case <-cctx.Done():
			goto wait
		case jobs <- pos:
		}
	}
wait:
	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// allPages returns [0, n) for the current page count, or an error if closed.
func (d *Document) allPages() ([]int, error) {
	n := d.PageCount()
	if n == 0 {
		if d.b == nil {
			return nil, errors.New("gomupdf: document closed")
		}
		return nil, nil
	}
	pages := make([]int, n)
	for i := range pages {
		pages[i] = i
	}
	return pages, nil
}

// TextByPageConcurrent is the parallel form of TextByPage: it extracts every
// page's reading-order text across a worker pool and returns one entry per page
// in order. See MapPages for cancellation and worker-count control.
func (d *Document) TextByPageConcurrent(ctx context.Context, opts ...PoolOption) ([]string, error) {
	pages, err := d.allPages()
	if err != nil {
		return nil, err
	}
	return MapPages(ctx, d, pages, (*Page).GetText, opts...)
}

// RenderPages renders the given pages concurrently and returns their pixmaps in
// input order — "fan a large PDF across all your cores" in one call. Pass the
// page indices to render (use a 0..N-1 slice for the whole document) and a
// single PixmapOptions applied to every page. See MapPages for cancellation and
// worker-count control.
func (d *Document) RenderPages(ctx context.Context, pages []int, opts PixmapOptions, poolOpts ...PoolOption) ([]*Pixmap, error) {
	return MapPages(ctx, d, pages, func(p *Page) (*Pixmap, error) {
		return p.Pixmap(opts)
	}, poolOpts...)
}
