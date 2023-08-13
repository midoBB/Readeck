package docs

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
)

//go:embed assets assets/**/* licenses/*
var assets embed.FS

// Files contains all the generated help files as an http.FS instance
var Files http.FileSystem

type File struct {
	Route      string         `json:"route"`
	Aliases    []string       `json:"aliases"`
	File       string         `json:"file"`
	Etag       string         `json:"etag"`
	IsDocument bool           `json:"is_document"`
	Title      string         `json:"title"`
	Meta       map[string]any `json:"meta"`
}

type Section struct {
	Files map[string]*File `json:"files"`
	TOC   [][2]string      `json:"toc"`
}

type Manifest struct {
	Files    map[string]*File    `json:"files"`
	Sections map[string]*Section `json:"sections"`
}

var manifest *Manifest

func (f *File) GetSumStrings() []string {
	return []string{f.Etag}
}

func init() {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	Files = http.FS(sub)

	// Load manifest
	fd, err := assets.Open("assets/manifest.json")
	if err != nil {
		panic(err)
	}

	dec := json.NewDecoder(fd)
	err = dec.Decode(&manifest)
	if err != nil {
		panic(err)
	}
}
