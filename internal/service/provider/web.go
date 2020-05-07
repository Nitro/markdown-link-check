package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

type webClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type webClientTransport struct {
	client webClient
}

func (w webClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return w.client.Do(req)
}

// Web handle the verification of HTTP endpoints.
type Web struct {
	client webClient
	regex  regexp.Regexp
}

// Init internal state.
func (w *Web) Init() error {
	if err := w.initRegex(); err != nil {
		return fmt.Errorf("fail to initialize the regex: %w", err)
	}
	w.initHTTP()
	return nil
}

// Authority checks if the web provider is responsible to process the entry.
func (w Web) Authority(uri string) bool {
	return w.regex.Match([]byte(uri))
}

// Valid check if the link is valid.
func (w Web) Valid(ctx context.Context, _, uri string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return false, fmt.Errorf("fail to create the HTTP request: %w", err)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	endpoint, err := url.Parse(uri)
	if err != nil {
		return false, fmt.Errorf("fail to parse uri: %w", err)
	}

	isValid := ((resp.StatusCode >= 200) && (resp.StatusCode < 300))
	if !isValid {
		return false, nil
	}

	validAnchor, err := w.validAnchor(resp.Body, endpoint.Fragment)
	if err != nil {
		return false, fmt.Errorf("fail to verify the anchor: %w", err)
	}

	return validAnchor, nil
}

func (w Web) validAnchor(body io.Reader, anchor string) (bool, error) {
	if anchor == "" {
		return true, nil
	}
	anchor = fmt.Sprintf("#%s", anchor)

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return false, fmt.Errorf("failt o parse the response: %w", err)
	}

	var found bool
	doc.Find("a").Each(func(_ int, selection *goquery.Selection) {
		if found {
			return
		}

		href, ok := selection.Attr("href")
		if !ok {
			return
		}
		found = (href == anchor)
	})

	return found, nil
}

func (w *Web) initRegex() error {
	expr := `^(http|https):\/\/`
	regex, err := regexp.Compile(expr)
	if err != nil {
		return fmt.Errorf("fail to compile the expression '%s': %w", expr, err)
	}
	w.regex = *regex
	return nil
}

func (w *Web) initHTTP() {
	if w.client == nil {
		w.client = &http.Client{}
	}

	w.client = &http.Client{
		Transport: webClientTransport{client: w.client},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			switch req.Response.StatusCode {
			case http.StatusPermanentRedirect, http.StatusMovedPermanently:
				return nil
			default:
				return errors.New("redirect not allowed")
			}
		},
	}
}
