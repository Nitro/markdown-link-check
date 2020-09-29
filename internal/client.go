package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/logrusorgru/aurora"

	"nitro/markdown-link-check/internal/service"
	"nitro/markdown-link-check/internal/service/parser"
	"nitro/markdown-link-check/internal/service/provider"
	"nitro/markdown-link-check/internal/service/scan"
	"nitro/markdown-link-check/internal/service/worker"
)

// ClientIgnore holds the ignore list for links and files.
type ClientIgnore struct {
	Link []string
	File []string
}

// ClientProviderGithub holds the configuration for the GitHub provider.
type ClientProviderGithub struct {
	Token      string
	Owner      string
	Repository string
}

// ClientProviderWeb holds the configuration for the web provider.
type ClientProviderWeb struct {
	Config          http.Header
	ConfigOverwrite map[string]http.Header
}

// ClientProvider holds the configuration for the providers.
type ClientProvider struct {
	Github []ClientProviderGithub
	Web    ClientProviderWeb
}

// Client is responsible to bootstrap the application.
type Client struct {
	Path     string
	Ignore   ClientIgnore
	Provider ClientProvider

	parser    parser.Markdown
	providers []worker.Provider
}

// Run starts the application execution.
func (c Client) Run(ctx context.Context) (bool, error) {
	if err := c.init(); err != nil {
		return false, fmt.Errorf("fail during init: %w", err)
	}

	s := scan.Scan{IgnoreFile: c.Ignore.File, IgnoreLink: c.Ignore.Link, Parser: c.parser}
	if err := s.Init(); err != nil {
		return false, fmt.Errorf("fail to initialize the scan service: %w", err)
	}
	entries, err := s.Process(c.Path)
	if err != nil {
		return false, fmt.Errorf("fail to scan the files: %w", err)
	}

	// entries = []service.Entry{
	// 	{
	// 		Path: "/Users/dbernardes/Source/Nitro/platform/engineering-documentation/deprecated/practices/account-links.md",
	// 		Link: "https://www.cloudflare.com/adasdasdadasdas",
	// 	},
	// }

	w := worker.Worker{Providers: c.providers}
	entries, err = w.Process(ctx, entries)
	if err != nil {
		return false, fmt.Errorf("fail to process the link: %w", err)
	}

	return c.output(entries), nil
}

func (c *Client) init() error {
	if c.Path == "" {
		return errors.New("missing 'path")
	}

	stat, err := os.Stat(c.Path)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %w", err)
	}
	if !stat.IsDir() {
		return errors.New("path is expected to be a directory")
	}

	var email provider.Email
	if err := email.Init(); err != nil {
		return fmt.Errorf("fail to iniitalize the email provider: %w", err)
	}
	c.providers = append(c.providers, email)

	for _, github := range c.Provider.Github {
		client := provider.GitHub{
			Token:      github.Token,
			Owner:      github.Owner,
			HTTPClient: http.DefaultClient,
		}
		if err := client.Init(); err != nil {
			return fmt.Errorf("fail to iniitalize the GitHub provider: %w", err)
		}
		c.providers = append(c.providers, client)
	}

	webConfigOverwrites := make(map[string]provider.WebConfig, len(c.Provider.Web.ConfigOverwrite))
	for key, value := range c.Provider.Web.ConfigOverwrite {
		webConfigOverwrites[key] = provider.WebConfig{Header: value}
	}
	w := provider.Web{
		Config:          provider.WebConfig{Header: c.Provider.Web.Config},
		ConfigOverwrite: webConfigOverwrites,
	}
	if err := w.Init(); err != nil {
		return fmt.Errorf("fail to initialize the web provider: %w", err)
	}
	c.providers = append(c.providers, w)

	var p parser.Markdown
	p.Init()
	c.parser = p
	f := provider.File{Path: c.Path, Parser: p}
	if err := f.Init(); err != nil {
		return fmt.Errorf("fail to initialize the file provider: %w", err)
	}
	c.providers = append(c.providers, f)

	return nil
}

func (c Client) output(entries []service.Entry) bool {
	var (
		result bool
		iter   = c.aggregate(entries)
	)
	for {
		key, entries, ok := iter()
		if !ok {
			break
		}
		if !c.hasInvalidLink(entries) {
			continue
		}
		if !result {
			result = true
		}

		fmt.Print(aurora.Bold(c.relativePath(key)))
		for _, entry := range entries {
			if entry.Valid {
				continue
			}
			fmt.Printf("\n%s %s", aurora.Bold(aurora.Gray(24, "-")), entry.Link)
		}
		fmt.Printf("\n\n")
	}

	// Printing the details of the failure.
	result = false
	iter = c.aggregate(entries)
	for {
		key, entries, ok := iter()
		if !ok {
			break
		}
		if !c.hasInvalidLink(entries) {
			continue
		}
		if !result {
			result = true
		}

		for _, entry := range entries {
			if entry.Valid || (entry.FailReason == nil) {
				continue
			}
			fmt.Printf(
				"The link '%s' at the file '%s' failed because of:\n",
				aurora.Bold(entry.Link), aurora.Bold(c.relativePath(key)),
			)
			entry.FailReason()
		}
		fmt.Printf("\n\n")
	}

	return result
}

func (Client) aggregate(entries []service.Entry) func() (string, []service.Entry, bool) {
	var (
		keys   = make([]string, 0, len(entries))
		result = make(map[string][]service.Entry)
	)

	for _, entry := range entries {
		if _, ok := result[entry.Path]; !ok {
			keys = append(keys, entry.Path)
			result[entry.Path] = make([]service.Entry, 0)
		}
		result[entry.Path] = append(result[entry.Path], entry)
	}
	sort.Strings(keys)

	var index = 0
	return func() (string, []service.Entry, bool) {
		if index >= len(keys) {
			return "", nil, false
		}
		key := keys[index]
		index++
		entries := result[key]
		sort.Sort(serviceEntrySort(entries))
		return key, entries, true
	}
}

func (Client) hasInvalidLink(entries []service.Entry) bool {
	for _, entry := range entries {
		if !entry.Valid {
			return true
		}
	}
	return false
}

func (c Client) relativePath(path string) string {
	dirPath := c.Path
	if !strings.HasSuffix(dirPath, "/") {
		dirPath += "/"
	}
	return strings.TrimPrefix(path, dirPath)
}

type serviceEntrySort []service.Entry

func (s serviceEntrySort) Len() int {
	return len(s)
}

func (s serviceEntrySort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s serviceEntrySort) Less(i, j int) bool {
	return s[i].Link < s[j].Link
}
