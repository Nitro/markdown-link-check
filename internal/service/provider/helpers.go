package provider

import (
	"io"
	"os"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

// FileHelpers contains the Helpers for the file provider
type FileHelpers interface {
	docQuery(io.Reader) (*goquery.Document, error)
	fileExists(string) (os.FileInfo, bool)
	readFile(string) ([]byte, error)
	regexCompile(string) (*regexp.Regexp, error)
}
