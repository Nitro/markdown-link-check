package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/logrusorgru/aurora"

	"nitro/markdown-link-check/internal/service"
	"nitro/markdown-link-check/internal/service/parser"
	"nitro/markdown-link-check/internal/service/provider"
	"nitro/markdown-link-check/internal/service/scan"
	"nitro/markdown-link-check/internal/service/worker"
)

// ClientProviderGithub holds the configuration for the GitHub provider.
type ClientProviderGithub struct {
	Token      string
	Owner      string
	Repository string
}

// ClientProvider holds the configuration for the providers.
type ClientProvider struct {
	Github []ClientProviderGithub
}

// Client is responsible to bootstrap the application.
type Client struct {
	Path     string
	Ignore   []string
	Provider ClientProvider

	parser    parser.Markdown
	providers []worker.Provider
}

// Run starts the application execution.
func (c Client) Run(ctx context.Context) (bool, error) {
	if err := c.init(); err != nil {
		return false, fmt.Errorf("fail during init: %w", err)
	}

	s := scan.Scan{Ignore: c.Ignore, Parser: c.parser}
	if err := s.Init(); err != nil {
		return false, fmt.Errorf("fail to initialize the scan service: %w", err)
	}
	entries, err := s.Process(c.Path)
	if err != nil {
		return false, fmt.Errorf("fail to scan the files: %w", err)
	}

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

	var p parser.Markdown
	p.Init()
	c.parser = p
	f := provider.File{Path: c.Path, Parser: p}
	if err := f.Init(); err != nil {
		return fmt.Errorf("fail to initialize the file provider: %w", err)
	}
	c.providers = append(c.providers, f)

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

		fmt.Print(aurora.Bold(aurora.Gray(24, "File: ")), key)
		for _, entry := range entries {
			if entry.Valid {
				continue
			}
			fmt.Printf("\n%s %s", aurora.Bold(aurora.Gray(24, "-")), entry.Link)
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
