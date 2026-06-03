package gomupdf

import "errors"

// HTML returns the page's structured text as HTML (styled, absolute-positioned).
// The output mirrors PyMuPDF's page.get_text("html").
func (p *Page) HTML() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.htmlRaw(p.Number)
}

// XHTML returns the page's structured text as semantic XHTML.
// The output mirrors PyMuPDF's page.get_text("xhtml").
func (p *Page) XHTML() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.xhtmlRaw(p.Number)
}

// XML returns the page's structured text as XML with per-character detail.
// The output mirrors PyMuPDF's page.get_text("xml").
func (p *Page) XML() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.xmlRaw(p.Number)
}
