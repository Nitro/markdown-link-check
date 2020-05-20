package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type fileReader interface {
	fileExists(string) (os.FileInfo, bool)
	readFile(string) ([]byte, error)
}

type fileParser interface {
	Do(payload []byte) []byte
	SanitizedAnchorName(text string) string
}

type fileReaderAPI struct{}

func (fileReaderAPI) fileExists(item string) (os.FileInfo, bool) {
	info, err := os.Stat(item)
	return info, !os.IsNotExist(err)
}

func (fileReaderAPI) readFile(filer string) ([]byte, error) {
	return ioutil.ReadFile(filer)
}

// File provider is responsible for checking if the file exists at the filesystem.
type File struct {
	Path   string
	Parser fileParser

	reader      fileReader
	schemaRegex regexp.Regexp
}

// Init internal state.
func (f *File) Init() error {
	if f.reader == nil {
		f.reader = fileReaderAPI{}
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
	if f.isMarkdown(filePath) {
		found, err := f.checkMarkdown(filePath, uri)
		if err != nil {
			return false, fmt.Errorf("fail to check the markdown: %w", err)
		}
		return found, nil
	}

	path := filepath.Join(filepath.Dir(filePath), uri)
	_, itemExists := f.reader.fileExists(path)
	return itemExists, nil
}

func (f *File) initRegex() error {
	expr := "^.*$"
	schema, err := regexp.Compile(expr)
	if err != nil {
		return fmt.Errorf("fail to compile the expression '%s': %w", expr, err)
	}
	f.schemaRegex = *schema
	return nil
}

func (File) isMarkdown(path string) bool {
	return filepath.Ext(path) == ".md"
}

// checkMarkdown check if the uri is a Markdown, if positive, it will be responsible to detect if the link is valid.
func (f File) checkMarkdown(path, uri string) (bool, error) {
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
	pathStat, valid := f.reader.fileExists(expandedPath)
	if !valid {
		return false, nil
	}
	if pathStat.IsDir() {
		return true, nil
	}

	if filepath.Ext(expandedPath) != ".md" {
		return true, nil
	}

	payload, err := f.reader.readFile(expandedPath)
	if err != nil {
		return false, fmt.Errorf("fail to read the file '%s': %w", expandedPath, err)
	}
	payload = f.Parser.Do(payload)

	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(payload))
	if err != nil {
		return false, fmt.Errorf("fail to parse the HTML: %w", err)
	}

	// The anchors are generated as links on the HTML. Here we'll look for the link, if we found it, it's a valid
	// response.
	var (
		found    bool
		fragment = f.Parser.SanitizedAnchorName(parsedURI.Fragment)
	)
	for i := 1; (i <= 6) && (!found); i++ {
		doc.Find(fmt.Sprintf("h%d", i)).Each(func(i int, selection *goquery.Selection) {
			if found {
				return
			}

			id, ok := selection.Attr("id")
			if !ok {
				return
			}

			// Check if the frament is in the id, this is for normal links.
			found = (strings.ToLower(id) == fragment)
			if found {
				return
			}

			// The fragment can point to a link as well, on this case we need to check if there is a link inside the h tag
			// with the fragment.
			selection.Each(func(_ int, selection *goquery.Selection) {
				if found {
					return
				}
				found = (strings.ToLower(selection.Text()) == fragment)
			})
		})
	}

	return found, nil
}
