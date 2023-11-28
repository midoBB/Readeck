// SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// tools/ftr provides a command line interface to convert site config text files
// to JSON files.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	flag.Parse()

	if len(flag.Args()) < 2 {
		log.Fatalf("Usage: fftr_convert <src> <dest>")
	}
	srcDir, _ := filepath.Abs(flag.Arg(0))
	destDir, _ := filepath.Abs(flag.Arg(1))

	info, err := os.Stat(srcDir)
	if err != nil {
		log.Fatal(err)
	}
	if !info.IsDir() {
		log.Fatalf("%s is not a directory", srcDir)
	}

	info, err = os.Stat(destDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if e := os.MkdirAll(destDir, 0o755); e != nil {
				log.Fatal(e)
			}
		} else {
			log.Fatal(err)
		}
	}
	if info != nil && !info.IsDir() {
		log.Fatalf("%s is not a directory", destDir)
	}

	log.Printf("Reading FiveFilters files from %s", srcDir)
	log.Printf("Destination: %s", destDir)

	// Parse fftr files
	err = filepath.Walk(srcDir, func(name string, info os.FileInfo, _ error) error {
		if path.Base(name) == "LICENSE.txt" {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if path.Ext(name) != ".txt" {
			return nil
		}

		if err := converTextConfig(name, destDir); err != nil {
			log.Printf("ERR: %s\n           %s", name, err)
			return nil
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

// Config holds the fivefilters configuration.
type Config struct {
	Files []string `json:"-"`

	TitleSelectors          []string          `json:"title_selectors"`
	BodySelectors           []string          `json:"body_selectors"`
	DateSelectors           []string          `json:"date_selectors"`
	AuthorSelectors         []string          `json:"author_selectors"`
	StripSelectors          []string          `json:"strip_selectors"`
	StripIDOrClass          []string          `json:"strip_id_or_class"`
	StripImageSrc           []string          `json:"strip_image_src"`
	NativeAdSelectors       []string          `json:"native_ad_selectors"`
	Tidy                    bool              `json:"tidy"`
	Prune                   bool              `json:"prune"`
	AutoDetectOnFailure     bool              `json:"autodetect_on_failure"`
	SinglePageLinkSelectors []string          `json:"single_page_link_selectors"`
	NextPageLinkSelectors   []string          `json:"next_page_link_selectors"`
	ReplaceStrings          [][2]string       `json:"replace_strings"`
	HTTPHeaders             map[string]string `json:"http_headers"`
	Tests                   []FilterTest      `json:"tests"`
}

// FilterTest holds the values for a filter's test.
type FilterTest struct {
	URL      string   `json:"url"`
	Contains []string `json:"contains"`
}

func converTextConfig(filename string, dest string) error {
	fp, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close() //nolint:errcheck

	cfg, err := newConfig(fp)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(cfg); err != nil {
		return err
	}

	destFile := path.Join(dest, path.Base(filename))
	destFile = destFile[0:len(destFile)-len(path.Ext(destFile))] + ".json"
	fd, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer fd.Close() //nolint:errcheck
	if _, err = fd.Write(buf.Bytes()); err != nil {
		return err
	}
	log.Printf("ok: %s", destFile)
	return nil
}

func newConfig(file io.Reader) (*Config, error) {
	res := &Config{
		AutoDetectOnFailure: true,
	}

	scanner := bufio.NewScanner(file)
	entries := make([][3]string, 0)
	for scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "//") {
			continue
		}
		entry, err := parseLine(t)
		if err != nil {
			return res, err
		}
		entries = append(entries, entry)
	}

	parseFunctions := map[string]entryParser{
		"body":                  simpleStringValue(&res.BodySelectors),
		"title":                 simpleStringValue(&res.TitleSelectors),
		"date":                  simpleStringValue(&res.DateSelectors),
		"author":                simpleStringValue(&res.AuthorSelectors),
		"strip":                 simpleStringValue(&res.StripSelectors),
		"strip_id_or_class":     simpleStringValue(&res.StripIDOrClass),
		"strip_image_src":       simpleStringValue(&res.StripImageSrc),
		"native_ad_clue":        simpleStringValue(&res.NativeAdSelectors),
		"prune":                 simpleBoolValue(&res.Prune),
		"tidy":                  simpleBoolValue(&res.Tidy),
		"autodetect_on_failure": simpleBoolValue(&res.AutoDetectOnFailure),
		"single_page_link":      simpleStringValue(&res.SinglePageLinkSelectors),
		"next_page_link":        simpleStringValue(&res.NextPageLinkSelectors),
		"http_header":           setHeaderValue,
		"find_string":           setReplaceString,
		"replace_string":        setReplaceString,
		"test_url":              setFilterTest,
	}

	for i, line := range entries {
		fn, ok := parseFunctions[line[0]]
		if ok {
			err := fn(res, i, entries)
			if err != nil {
				return res, err
			}
		}
	}

	return res, nil
}

var lineRE = regexp.MustCompile(`^(.+?)(?:\((.+)\))?:\s*(.*)$`)

func parseLine(line string) ([3]string, error) {
	if !lineRE.MatchString(line) {
		return [3]string{}, fmt.Errorf("Cannot parse line (%s)", line)
	}

	m := lineRE.FindAllStringSubmatch(line, -1)
	if strings.HasPrefix(m[0][3], "'") && strings.HasSuffix(m[0][3], "'") && len(m[0][3]) > 1 {
		m[0][3] = m[0][3][1 : len(m[0][3])-1]
	}

	return [3]string{m[0][1], m[0][2], m[0][3]}, nil
}

type entryParser func(*Config, int, [][3]string) error

func simpleStringValue(v *[]string) entryParser {
	return func(cfg *Config, i int, entries [][3]string) error {
		*v = append(*v, entries[i][2])
		return nil
	}
}

func simpleBoolValue(v *bool) entryParser {
	return func(cfg *Config, i int, entries [][3]string) error {
		*v = entries[i][2] == "yes"
		return nil
	}
}

func setHeaderValue(cfg *Config, i int, entries [][3]string) error {
	if entries[i][1] == "" {
		return fmt.Errorf("Header value not set (%s)", entries[i][2])
	}

	if cfg.HTTPHeaders == nil {
		cfg.HTTPHeaders = map[string]string{}
	}
	cfg.HTTPHeaders[entries[i][1]] = entries[i][2]
	return nil
}

func setReplaceString(cfg *Config, i int, entries [][3]string) error {
	line := entries[i]
	switch line[0] {
	case "replace_string":
		if line[1] != "" {
			cfg.ReplaceStrings = append(cfg.ReplaceStrings, [2]string{line[1], line[2]})
			return nil
		}
		if i-1 < 0 {
			return fmt.Errorf("No preceding find_string entry before replace_string: %s", line[2])
		}
		prev := entries[i-1]
		if prev[0] != "find_string" {
			return fmt.Errorf("Invalid preceding entry before replace_string: %s", line[2])
		}
	case "find_string":
		if i+1 >= len(entries) {
			return fmt.Errorf("No subsequent replace_string entry after find_string: %s", line[2])
		}
		next := entries[i+1]
		if next[0] != "replace_string" {
			return fmt.Errorf("Invalid subsequent entry after find_string: %s", line[2])
		}
		if next[1] != "" {
			return fmt.Errorf("Invalid subsequent entry after find_string: %s", line[2])
		}
		cfg.ReplaceStrings = append(cfg.ReplaceStrings, [2]string{line[2], next[2]})
	}
	return nil
}

func setFilterTest(cfg *Config, i int, entries [][3]string) error {
	line := entries[i]
	res := FilterTest{URL: line[2], Contains: make([]string, 0)}

	for {
		i++
		if i < len(entries) && entries[i][0] == "test_contains" {
			res.Contains = append(res.Contains, entries[i][2])
			continue
		}
		break
	}
	cfg.Tests = append(cfg.Tests, res)
	return nil
}
