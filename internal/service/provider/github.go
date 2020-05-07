package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
)

type gitHubRepository interface {
	repository(ctx context.Context, repository string) (*github.Response, error)
	repositoriesGetCommitSHA1(ctx context.Context, repository, ref string) (*github.Response, error)
	issuesGet(ctx context.Context, repo string, number int) (*github.Response, error)
	issuesGetComment(ctx context.Context, repo string, commentID int64) (*github.Response, error)
	pullRequestsGetRaw(ctx context.Context, repo string, number int) (*github.Response, error)
	relatedPullRequests(ctx context.Context, repository, ref string) ([]int, error)
}

type gitHubHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// GitHub provider.
type GitHub struct {
	HTTPClient gitHubHTTPClient
	Token      string
	Owner      string

	repository       gitHubRepository
	regexOwner       regexp.Regexp
	regexRepository  regexp.Regexp
	regexRaw         regexp.Regexp
	regexBase        regexp.Regexp
	regexCommit      regexp.Regexp
	regexIssue       regexp.Regexp
	regexPullRequest regexp.Regexp
}

// Init the internal state.
func (g *GitHub) Init() error {
	if g.HTTPClient == nil {
		return errors.New("missing 'httpClient")
	}

	if g.repository == nil {
		api := githubAPI{token: g.Token, owner: g.Owner, httpClient: g.HTTPClient}
		if err := api.init(); err != nil {
			return fmt.Errorf("fail to initialize the GitHub client: %w", err)
		}
		g.repository = api
	}

	if err := g.initRegex(); err != nil {
		return fmt.Errorf("fail to initialize the regex expressions: %w", err)
	}

	return nil
}

// Authority checks if the github provider is responsible to process the entry.
func (g GitHub) Authority(uri string) bool {
	for _, expr := range []regexp.Regexp{g.regexRaw, g.regexBase} {
		if expr.Match([]byte(uri)) {
			return true
		}
	}
	return false
}

// Valid check if the link is valid.
func (g GitHub) Valid(ctx context.Context, _, uri string) (bool, error) {
	fns := []func(context.Context, string) (bool, error){
		g.validOwner,
		g.validCommit,
		g.validIssue,
		g.validPullRequest,
		g.validRepository,
	}
	for _, fn := range fns {
		valid, err := fn(ctx, uri)
		if err != nil {
			return false, nil
		}
		if valid {
			return true, nil
		}
	}
	return false, nil
}

func (g *GitHub) initRegex() error {
	compile := func(rawExpr string) (regexp.Regexp, error) {
		expr, err := regexp.Compile(rawExpr)
		if err != nil {
			return regexp.Regexp{}, fmt.Errorf("fail to compile the expression '%s': %w", rawExpr, err)
		}
		return *expr, nil
	}

	var err error
	regexRaw := fmt.Sprintf(
		`^(?P<schema>http|https):\/\/raw\.githubusercontent\.com\/((?i)%s)`,
		regexp.QuoteMeta(g.Owner),
	)
	g.regexRaw, err = compile(regexRaw)
	if err != nil {
		return err
	}

	regexBase := fmt.Sprintf(`^(?P<schema>http|https):\/\/github\.com\/((?i)%s)`, g.Owner)
	g.regexBase, err = compile(regexBase)
	if err != nil {
		return err
	}

	regexOwner := fmt.Sprintf(`^(?P<schema>http|https):\/\/github\.com\/((?i)%s)$`, g.Owner)
	g.regexOwner, err = compile(regexOwner)
	if err != nil {
		return err
	}

	regexCommit := fmt.Sprintf(
		`^(?P<schema>http|https):\/\/github\.com\/((?i)%s)\/(?P<repository>.*)\/commit\/(?P<commit>.*)$`, g.Owner,
	)
	g.regexCommit, err = compile(regexCommit)
	if err != nil {
		return err
	}

	regexRepository := fmt.Sprintf(`^(?P<schema>http|https):\/\/github\.com\/((?i)%s)\/(?P<repository>.*)$`, g.Owner)
	g.regexRepository, err = compile(regexRepository)
	if err != nil {
		return err
	}

	var regexIssue strings.Builder
	fmt.Fprintf(&regexIssue, `^(?P<schema>http|https):\/\/github\.com\/((?i)%s)\/(?P<repository>.*)\/issues\/`, g.Owner)
	fmt.Fprint(&regexIssue, `(?P<issueID>[0-9]*)(#issuecomment-(?P<commentID>[0-9]*))?$`)
	g.regexIssue, err = compile(regexIssue.String())
	if err != nil {
		return err
	}

	var regexPullRequest strings.Builder
	fmt.Fprintf(
		&regexPullRequest, `^(?P<schema>http|https):\/\/github\.com\/((?i)%s)\/(?P<repository>.*)\/pull\/`, g.Owner,
	)
	fmt.Fprint(&regexPullRequest, `(?P<pullID>[0-9]*)(\/commits\/(?P<ref>.*))?$`)
	g.regexPullRequest, err = compile(regexPullRequest.String())
	if err != nil {
		return err
	}

	return nil
}

// validOwner is not doing the complete verification as it's not available at the API. We're trusting that the owner
// exists based on the configuration the client provided.
func (g GitHub) validOwner(ctx context.Context, uri string) (bool, error) {
	fragments := g.regexOwner.FindStringSubmatch(uri)
	if fragments == nil {
		return false, nil
	}
	return true, nil
}

func (g GitHub) validRepository(ctx context.Context, uri string) (bool, error) {
	fragments := g.regexRepository.FindStringSubmatch(uri)
	if fragments == nil {
		return false, nil
	}

	resp, err := g.repository.repository(ctx, fragments[3])
	if err != nil {
		return false, fmt.Errorf("fail to consult the repository: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil
}

func (g GitHub) validCommit(ctx context.Context, uri string) (bool, error) {
	fragments := g.regexCommit.FindStringSubmatch(uri)
	if fragments == nil {
		return false, nil
	}

	resp, err := g.repository.repositoriesGetCommitSHA1(ctx, fragments[3], fragments[4])
	if err != nil {
		return false, fmt.Errorf("fail to consult the commit at GitHub: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, nil
}

func (g GitHub) validIssue(ctx context.Context, uri string) (bool, error) {
	fragments := g.regexIssue.FindStringSubmatch(uri)
	if fragments == nil {
		return false, nil
	}

	issue, err := strconv.ParseInt(fragments[4], 10, 64)
	if err != nil {
		return false, fmt.Errorf("fail to parse the issue value: %w", err)
	}

	resp, err := g.repository.issuesGet(ctx, fragments[3], (int)(issue))
	if err != nil {
		return false, fmt.Errorf("fail to consult the issue at GitHub: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	if fragments[6] == "" {
		return true, nil
	}

	return g.validIssueComment(ctx, fragments)
}

func (g GitHub) validIssueComment(ctx context.Context, fragments []string) (bool, error) {
	comment, err := strconv.ParseInt(fragments[6], 10, 64)
	if err != nil {
		return false, fmt.Errorf("fail to parse the comment value: %w", err)
	}

	resp, err := g.repository.issuesGetComment(ctx, fragments[3], comment)
	if err != nil {
		return false, fmt.Errorf("fail to consult the issue comment at GitHub: %w", err)
	}
	defer resp.Body.Close()
	return (resp.StatusCode == http.StatusOK), nil
}

func (g GitHub) validPullRequest(ctx context.Context, uri string) (bool, error) {
	fragments := g.regexPullRequest.FindStringSubmatch(uri)
	if fragments == nil {
		return false, nil
	}

	pullRequestID, err := strconv.ParseInt(fragments[4], 10, 64)
	if err != nil {
		return false, fmt.Errorf("fail to parse the pull request ID value: %w", err)
	}

	resp, err := g.repository.pullRequestsGetRaw(ctx, fragments[3], (int)(pullRequestID))
	if err != nil {
		return false, fmt.Errorf("fail to consult the pull request at GitHub: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	return g.validPullRequestCommit(ctx, (int)(pullRequestID), fragments)
}

func (g GitHub) validPullRequestCommit(ctx context.Context, pullRequestID int, fragments []string) (bool, error) {
	if fragments[6] == "" {
		return true, nil
	}

	pullRequestIDS, err := g.repository.relatedPullRequests(ctx, fragments[3], fragments[6])
	if err != nil {
		return false, fmt.Errorf("fail to fetch the pull requests associated with the commit: %w", err)
	}

	for _, id := range pullRequestIDS {
		if id == pullRequestID {
			return true, nil
		}
	}

	return false, nil
}
