package web

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed templates/*.html static/*
var files embed.FS

func Templates() (*template.Template, error) {
	return template.ParseFS(files, "templates/*.html")
}

func StaticFS() fs.FS {
	static, err := fs.Sub(files, "static")
	if err != nil {
		panic(err)
	}
	return static
}
