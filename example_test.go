package gomupdf_test

import (
	"fmt"
	"log"

	"github.com/srijanmukherjee/gomupdf"
)

// Open a PDF and print the reading-order text of every page.
func ExampleOpen() {
	doc, err := gomupdf.Open("document.pdf")
	if err != nil {
		log.Fatal(err)
	}
	defer doc.Close()

	for i, page := range doc.Pages() {
		text, err := page.GetText()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("page %d:\n%s\n", i, text)
	}
}

// Open an encrypted PDF, authenticating with a password.
func ExampleDocument_Authenticate() {
	doc, err := gomupdf.Open("encrypted.pdf")
	if err != nil {
		log.Fatal(err)
	}
	defer doc.Close()

	if doc.NeedsPass() && !doc.Authenticate("secret") {
		log.Fatal("wrong password")
	}

	fmt.Println(doc.PageCount())
}

// Render the first page to a PNG file at 144 DPI.
func ExamplePage_Pixmap() {
	doc, err := gomupdf.Open("document.pdf")
	if err != nil {
		log.Fatal(err)
	}
	defer doc.Close()

	page, err := doc.LoadPage(0)
	if err != nil {
		log.Fatal(err)
	}
	pm, err := page.Pixmap(gomupdf.PixmapOptions{Zoom: 2.0})
	if err != nil {
		log.Fatal(err)
	}
	if err := pm.SavePNG("page-0.png"); err != nil {
		log.Fatal(err)
	}
}
