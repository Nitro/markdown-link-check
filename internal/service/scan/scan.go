package scan

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/PuerkitoBio/goquery"

	"nitro/markdown-link-check/internal/service"
)

type scanParser interface {
	Do(payload []byte) []byte
}

// Scan is responsible for reading, parsing and extracting links from the markdown files.
type Scan struct {
	Ignore []string
	Parser scanParser

	regex []regexp.Regexp
}

// Init the internal state.
func (s *Scan) Init() error {
	if s.Parser == nil {
		return errors.New("missing 'parser'")
	}

	for _, ignore := range s.Ignore {
		regex, err := regexp.Compile(ignore)
		if err != nil {
			return fmt.Errorf("fail to compile regex '%s': %w", ignore, err)
		}
		s.regex = append(s.regex, *regex)
	}
	return nil
}

// Process the directory.
func (s Scan) Process(path string) ([]service.Entry, error) {
	if err := s.isDir(path); err != nil {
		return nil, fmt.Errorf("fail to check if path is directory: %w", err)
	}

	files, err := s.listFiles(path)
	if err != nil {
		return nil, fmt.Errorf("fail to fetch the markdown file: %w", err)
	}
	sort.Strings(files)

	var result []service.Entry
	for _, file := range files {
		entries, err := s.processFile(file)
		if err != nil {
			return nil, fmt.Errorf("fail to process the file '%s': %w", file, err)
		}
		result = append(result, entries...)
	}

	return result, nil
}

func (Scan) isDir(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("fail to check the path stat: %w", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("'%s' expected to be a directory", path)
	}
	return nil
}

func (Scan) listFiles(path string) ([]string, error) {
	var paths []string

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) != ".md" {
			return nil
		}

		paths = append(paths, path)
		return nil
	}
	if err := filepath.Walk(path, walkFn); err != nil {
		return nil, fmt.Errorf("fail to fetch the files paths: %w", err)
	}

	return paths, nil
}

func (s Scan) processFile(path string) ([]service.Entry, error) {
	payload, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fail to read the file: %w", err)
	}

	html := s.Parser.Do(payload)
	links, err := s.extractLinks(html)
	if err != nil {
		return nil, fmt.Errorf("fail to extract links: %w", err)
	}
	links = s.removeDuplicates(links)

	result := make([]service.Entry, 0, len(links))
	for _, link := range links {
		result = append(result, service.Entry{Path: path, Link: link})
	}

	return result, nil
}

func (s Scan) extractLinks(payload []byte) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("fail to parse the HTML: %w", err)
	}

	var links []string
	doc.Find("a").Each(func(i int, selection *goquery.Selection) {
		href, ok := selection.Attr("href")
		if !ok {
			return
		}

		for _, regex := range s.regex {
			if regex.Match([]byte(href)) {
				return
			}
		}

		links = append(links, href)
	})
	return links, nil
}

func (Scan) removeDuplicates(elements []string) []string {
	index := make(map[string]struct{})
	for v := range elements {
		index[elements[v]] = struct{}{}
	}

	result := make([]string, 0, len(index))
	for key := range index {
		result = append(result, key)
	}
	return result
}
