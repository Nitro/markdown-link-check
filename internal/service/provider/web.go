package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// Rod is very sensitive and for now, the best approach is to have a mutex at the package level protecting all the
// operations.
var webBrowserMutex sync.Mutex // nolint: gochecknoglobals

type webClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type webClientTransport struct {
	client webClient
}

func (w webClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return w.client.Do(req)
}

type webConfigRegex struct {
	expression regexp.Regexp
	key        string
}

// WebConfig has the information to enhance the request.
type WebConfig struct {
	Header http.Header
}

// Web handle the verification of HTTP endpoints.
type Web struct {
	Config          WebConfig
	ConfigOverwrite map[string]WebConfig

	browser              *rod.Browser
	client               webClient
	regex                regexp.Regexp
	regexConfigOverwrite []webConfigRegex
}

// Init internal state.
func (w *Web) Init() error {
	if err := w.initRegex(); err != nil {
		return fmt.Errorf("fail to initialize the regex: %w", err)
	}
	if err := w.initRegexConfig(); err != nil {
		return fmt.Errorf("fail to initialize the regex config: %w", err)
	}
	w.initHTTP()
	if err := w.initBrowser(); err != nil {
		return fmt.Errorf("failed to initialize the browser: %w", err)
	}
	return nil
}

// Close the provider.
func (w *Web) Close() error {
	if err := w.browser.CloseE(); err != nil {
		return fmt.Errorf("failed to close the browser: %w", err)
	}
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
	w.configRequest(req)

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
	if validAnchor {
		return true, nil
	}

	validAnchor, err = w.validAnchorBrowser(ctx, uri, endpoint.Fragment)
	if err != nil {
		return false, fmt.Errorf("fail to verify the anchor with a browser: %w", err)
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

func (w *Web) initRegexConfig() error {
	w.regexConfigOverwrite = make([]webConfigRegex, 0, len(w.regexConfigOverwrite))
	for key := range w.ConfigOverwrite {
		regex, err := regexp.Compile(key)
		if err != nil {
			return fmt.Errorf("fail to compile the expression '%s': %w", key, err)
		}
		w.regexConfigOverwrite = append(w.regexConfigOverwrite, webConfigRegex{key: key, expression: *regex})
	}
	return nil
}

func (w *Web) initHTTP() {
	w.client = &http.Client{
		Transport: webClientTransport{client: http.DefaultClient},
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

func (w *Web) initBrowser() error {
	webBrowserMutex.Lock()
	defer webBrowserMutex.Unlock()

	launcherURL, err := launcher.New().Headless(true).LaunchE()
	if err != nil {
		return fmt.Errorf("failed to launch the browser: %w", err)
	}

	w.browser = rod.New().ControlURL(launcherURL)
	if err = w.browser.ConnectE(); err != nil {
		return fmt.Errorf("failed to connect to the browser: %w", err)
	}
	return nil
}

func (w Web) validAnchorBrowser(ctx context.Context, endpoint string, anchor string) (_ bool, err error) {
	webBrowserMutex.Lock()
	defer webBrowserMutex.Unlock()

	pctx, pctxCancel := context.WithCancel(ctx)
	defer pctxCancel()

	page, err := w.browser.PageE("")
	if err != nil {
		return false, fmt.Errorf("failed to create the browser page: %w", err)
	}

	if _, err = page.Context(pctx, pctxCancel).SetExtraHeadersE(w.genHeaders(endpoint)); err != nil {
		return false, fmt.Errorf("failed to set the headers at the browser page: %w", err)
	}

	if err := page.NavigateE(endpoint); err != nil {
		return false, fmt.Errorf("failed to navigate to the page: %w", err)
	}

	if err := page.WaitLoadE(); err != nil {
		return false, fmt.Errorf("failed to wait for the page to load: %w", err)
	}
	defer func() {
		if perr := page.CloseE(); perr != nil {
			err = fmt.Errorf("failed to close the browser tab: %w", perr)
		}
	}()

	result, err := page.EvalE(true, "", "document.documentElement.innerHTML", nil)
	if err != nil {
		return false, fmt.Errorf("failed to execute the javascript at the page: %w", err)
	}
	return w.validAnchor(bytes.NewBufferString(result.Value.String()), anchor)
}

func (w Web) configRequest(r *http.Request) {
	setHeader := func(header http.Header) {
		for key, values := range header {
			for _, value := range values {
				r.Header.Set(key, value)
			}
		}
	}

	setHeader(w.Config.Header)

	endpoint := r.URL.String()
	for _, cfg := range w.regexConfigOverwrite {
		if cfg.expression.Match([]byte(endpoint)) {
			setHeader(w.ConfigOverwrite[cfg.key].Header)
			return
		}
	}
}

func (w Web) genHeaders(endpoint string) []string {
	header := make(http.Header)

	setHeader := func(source http.Header) {
		for key, values := range source {
			for _, value := range values {
				header.Set(key, value)
			}
		}
	}
	setHeader(w.Config.Header)

	for _, cfg := range w.regexConfigOverwrite {
		if cfg.expression.MatchString(endpoint) {
			setHeader(w.ConfigOverwrite[cfg.key].Header)
			break
		}
	}

	results := make([]string, 0, len(header)*2)
	for key := range header {
		results = append(results, key, header.Get(key))
	}
	return results
}
