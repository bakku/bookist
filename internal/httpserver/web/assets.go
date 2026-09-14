package web

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed templates static
var files embed.FS

type TemplateSet struct {
	BooksIndex *template.Template
}

func Templates() (TemplateSet, error) {
	base, err := template.ParseFS(
		files,
		"templates/components.html",
		"templates/icons.html",
		"templates/layout.html",
	)
	if err != nil {
		return TemplateSet{}, err
	}

	booksIndex, err := parsePage(base, "templates/books/index.html")
	if err != nil {
		return TemplateSet{}, err
	}

	return TemplateSet{BooksIndex: booksIndex}, nil
}

func parsePage(base *template.Template, filename string) (*template.Template, error) {
	page, err := base.Clone()
	if err != nil {
		return nil, err
	}

	return page.ParseFS(files, filename)
}

func StaticFS() fs.FS {
	static, err := fs.Sub(files, "static")
	if err != nil {
		panic(err)
	}
	return static
}
