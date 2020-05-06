package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

type fileParser interface {
	Do(payload []byte) []byte
}

// fileHelpersC contain the implemenation of helpers for providers
type fileHelpersC struct{}

// File provider is responsible for checking if the file exists at the filesystem.
type File struct {
	Path    string
	Parser  fileParser
	Helpers FileHelpers

	schemaRegex regexp.Regexp
}

// Helpers
func (h fileHelpersC) fileExists(item string) (os.FileInfo, bool) {
	finfo, err := os.Stat(item)
	if os.IsNotExist(err) {
		return finfo, false
	}
	return finfo, true
}

func (h fileHelpersC) regexCompile(str string) (*regexp.Regexp, error) {
	return regexp.Compile(str)
}

func (h fileHelpersC) readFile(filer string) ([]byte, error) {
	return ioutil.ReadFile(filer)
}

func (h fileHelpersC) docQuery(r io.Reader) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(r)
}

// Init internal state.
func (f *File) Init() error {
	// If not Helpers assigned: use default
	if f.Helpers == nil {
		f.Helpers = fileHelpersC{}
	}

	if f.Path == "" {
		return errors.New("missing 'path'")
	}

	if f.Parser == nil {
		return errors.New("missing 'parser'")
	}

	if err := f.initRegex(); err != nil {
		return fmt.Errorf("fail to initialize the regex expressions: %w", err)
	}

	return nil
}

// Authority checks if the file provider is responsible to process the entry.
func (f File) Authority(uri string) bool {
	return f.schemaRegex.Match([]byte(uri))
}

// Valid check if the link is valid.
func (f File) Valid(ctx context.Context, filePath, uri string) (bool, error) {
	found, err := f.checkMarkdown(filePath, uri)
	if err != nil {
		return false, fmt.Errorf("fail to check the markdown: %w", err)
	}
	if found {
		return true, nil
	}

	path := filepath.Join(filepath.Dir(filePath), uri)
	_, itemExists := f.Helpers.fileExists(path)
	return itemExists, nil
}

func (f *File) initRegex() error {
	expr := "^.*$"
	schema, err := f.Helpers.regexCompile(expr)
	if err != nil {
		return fmt.Errorf("fail to compile the expression '%s': %w", expr, err)
	}
	f.schemaRegex = *schema
	return nil
}

// checkMarkdown check if the uri is a Markdown, if positive, it will be responsible to detect if the link is valid.
func (f File) checkMarkdown(path, uri string) (bool, error) {

	if filepath.Ext(path) != ".md" {
		return false, nil
	}

	parsedURI, err := url.Parse(uri)
	if err != nil {
		return false, fmt.Errorf("fail to parse the uri '%s': %w", uri, err)
	}

	// If the link is just a anchor like '#something' it will fit into the first condition. Otherwise it will be something
	// like this 'file.md' or 'file.md#something' and it will fall into the second condition.
	var expandedPath string
	if parsedURI.Path == "" {
		expandedPath = path
	} else {
		expandedPath = filepath.Join(filepath.Dir(path), parsedURI.Path)
	}

	// Check if the path exists, if not, it will return to the fallback verification at the caller.
	// If the path is a directory we get a valid response.
	pathStat, valid := f.Helpers.fileExists(expandedPath)

	if !valid {
		return false, nil
	}

	if pathStat.IsDir() {
		return true, nil
	}

	payload, err := f.Helpers.readFile(expandedPath)
	if err != nil {
		return false, fmt.Errorf("fail to read the file '%s': %w", expandedPath, err)
	}
	payload = f.Parser.Do(payload)

	doc, err := f.Helpers.docQuery(bytes.NewBuffer(payload))
	if err != nil {
		return false, fmt.Errorf("fail to parse the HTML: %w", err)
	}

	// The anchors are generated as links on the HTML. Here we'll look for the link, if we found it, it's a valid
	// response.
	var found bool
	doc.Find("a").Each(func(i int, selection *goquery.Selection) {
		if found {
			return
		}

		href, ok := selection.Attr("href")
		if !ok {
			return
		}

		rel, ok := selection.Attr("rel")
		if !ok {
			return
		}

		found = ((href == uri) && (rel == "nofollow"))
	})

	return found, nil
}
