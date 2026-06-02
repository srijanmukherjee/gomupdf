package experimental

// Info is a typed snapshot of document-level facts, sparing callers the
// stringly-typed metadata map and separate count/encryption calls.
type Info struct {
	Pages     int
	Encrypted bool
	Title     string
	Author    string
	Subject   string
	Keywords  string
	Creator   string
	Producer  string
	Format    string
	Created   string // raw creationDate string
	Modified  string // raw modDate string
	Metadata  map[string]string
}

// Info returns a typed summary of the document.
func (d *Doc) Info() (Info, error) {
	meta, err := d.raw.Metadata()
	if err != nil {
		return Info{}, err
	}
	return Info{
		Pages:     d.raw.PageCount(),
		Encrypted: d.raw.IsEncrypted(),
		Title:     meta["title"],
		Author:    meta["author"],
		Subject:   meta["subject"],
		Keywords:  meta["keywords"],
		Creator:   meta["creator"],
		Producer:  meta["producer"],
		Format:    meta["format"],
		Created:   meta["creationDate"],
		Modified:  meta["modDate"],
		Metadata:  meta,
	}, nil
}

// Bookmark is one entry of the document outline (table of contents).
type Bookmark struct {
	Level int // 1-based nesting depth
	Title string
	Page  int // 0-based target page, or -1 if external/unresolved
}

// Outline returns the document's table of contents as a flat, depth-first list.
func (d *Doc) Outline() ([]Bookmark, error) {
	toc, err := d.raw.TOC()
	if err != nil {
		return nil, err
	}
	out := make([]Bookmark, len(toc))
	for i, e := range toc {
		out[i] = Bookmark{Level: e.Level, Title: e.Title, Page: e.Page}
	}
	return out, nil
}

// Link is a clickable link on a page.
type Link struct {
	Rect Rect
	URI  string
}

// Links returns the page's clickable links.
func (p *Page) Links() ([]Link, error) {
	raw, err := p.raw.Links()
	if err != nil {
		return nil, err
	}
	out := make([]Link, len(raw))
	for i, l := range raw {
		out[i] = Link{Rect: rectFromGeometry(l.Rect), URI: l.URI}
	}
	return out, nil
}
