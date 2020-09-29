package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"nitro/markdown-link-check/internal/service"

	"github.com/logrusorgru/aurora"
)

// Provider represents the providers resonsible to process the entries.
type Provider interface {
	Authority(uri string) bool
	Valid(ctx context.Context, filePath, uri string) (bool, error)
} // nolint: golint

type workerError struct {
	units []workerErrorUnit
}

func (w workerError) Error() string {
	if len(w.units) == 1 {
		return w.units[0].err.Error()
	}

	errors := make([]string, 0, len(w.units))
	for _, unit := range w.units {
		errors = append(errors, unit.err.Error())
	}
	return fmt.Sprintf("multiple errors detected ('%s')", strings.Join(errors, "', '"))
}

type workerErrorUnit struct {
	err   error
	entry service.Entry
}

// Worker process the entries to check if they're valid. Everything is basead on providers and they're executed in
// order.
type Worker struct {
	Providers []Provider
}

// Process the entries.
func (w Worker) Process(ctx context.Context, entries []service.Entry) ([]service.Entry, error) {
	if len(w.Providers) == 0 {
		return nil, errors.New("missing 'providers'")
	}

	var (
		errors    []workerErrorUnit
		result    []service.Entry
		processed int
		total     = len(entries)
	)

	for _, entry := range entries {
		for _, provider := range w.Providers {
			if !provider.Authority(entry.Link) {
				continue
			}

			valid, err := provider.Valid(ctx, entry.Path, entry.Link)
			if e, ok := err.(service.EnhancedError); err != nil && !ok {
				errors = append(errors, workerErrorUnit{err: err, entry: entry})
			} else {
				if ok {
					entry.FailReason = e.PrettyPrint
				}
				entry.Valid = valid
				result = append(result, entry)
			}
			processed++
			fmt.Printf("%d of %d entries processed\n", aurora.Bold(processed), aurora.Bold(total))
			break
		}
	}

	if len(errors) == 0 {
		return result, nil
	}
	return nil, workerError{units: errors}
}
