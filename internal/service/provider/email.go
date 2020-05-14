package provider

import (
	"context"
	"fmt"
	"net"
	"regexp"
)

type emailChecker interface {
	exists(domain string) (bool, error)
}

// Email handles the verification of email addresses.
type Email struct {
	checker emailChecker
	regex   regexp.Regexp
}

// Init internal state.
func (e *Email) Init() error {
	if err := e.initRegex(); err != nil {
		return fmt.Errorf("fail to initialize the regex: %w", err)
	}

	if e.checker == nil {
		e.checker = emailNetLookupMX{}
	}

	return nil
}

// Authority checks if the email provider is responsible to process the entry.
func (e Email) Authority(uri string) bool {
	return e.regex.Match([]byte(uri))
}

// Valid check if the address is valid.
func (e Email) Valid(ctx context.Context, _, uri string) (bool, error) {
	fragments := e.regex.FindStringSubmatch(uri)
	if fragments == nil {
		return false, nil
	}

	exists, err := e.checker.exists(fragments[1])
	if err != nil {
		return false, fmt.Errorf("fail to check the MX DNS entries: %w", err)
	}
	return exists, nil
}

func (e *Email) initRegex() error {
	rawExpr := "^mailto:[a-zA-Z0-9.!#$%&’*+/=?^_`{|}~-]+@(?P<domain>[a-zA-Z0-9-]+(?:\\.[a-zA-Z0-9-]+)*)"
	expr, err := regexp.Compile(rawExpr)
	if err != nil {
		return fmt.Errorf("fail to compile the expression '%s': %w", rawExpr, err)
	}
	e.regex = *expr
	return nil
}

type emailNetLookupMX struct{}

func (emailNetLookupMX) exists(domain string) (bool, error) {
	mxs, err := net.LookupMX(domain)
	if err != nil {
		return false, err
	}
	return (len(mxs) > 0), nil
}
