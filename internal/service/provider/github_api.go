package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type githubAPI struct {
	token      string
	owner      string
	client     *github.Client
	httpClient gitHubHTTPClient
}

func (g *githubAPI) init() error {
	if g.token == "" {
		return errors.New("missing 'token'")
	}

	if g.owner == "" {
		return errors.New("missing 'owner'")
	}

	if g.httpClient == nil {
		return errors.New("missing 'httpClient")
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: g.token})
	tc := oauth2.NewClient(context.Background(), ts)
	g.client = github.NewClient(tc)
	return nil
}

func (g githubAPI) repository(ctx context.Context, repository string) (*github.Response, error) {
	_, resp, err := g.client.Repositories.Get(ctx, g.owner, repository)
	return resp, err
}

func (g githubAPI) repositoriesGetCommitSHA1(ctx context.Context, repository, ref string) (*github.Response, error) {
	_, resp, err := g.client.Repositories.GetCommitSHA1(ctx, g.owner, repository, ref, "")
	return resp, err
}

func (g githubAPI) issuesGet(ctx context.Context, repo string, number int) (*github.Response, error) {
	_, resp, err := g.client.Issues.Get(ctx, g.owner, repo, number)
	return resp, err
}

func (g githubAPI) issuesGetComment(ctx context.Context, repo string, commentID int64) (*github.Response, error) {
	_, resp, err := g.client.Issues.GetComment(ctx, g.owner, repo, commentID)
	return resp, err
}

func (g githubAPI) pullRequestsGetRaw(ctx context.Context, repo string, number int) (*github.Response, error) {
	_, resp, err := g.client.PullRequests.GetRaw(ctx, g.owner, repo, number, github.RawOptions{Type: github.Patch})
	return resp, err
}

// RelatedPullRequests is at developer preview and maybe this is the reason it's not available to be used though the
// GitHub client.
//
// For more information: https://developer.github.com/v3/repos/commits/#list-pull-requests-associated-with-commit
func (g githubAPI) relatedPullRequests(ctx context.Context, repository, ref string) ([]int, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s/pulls", g.owner, repository, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("fail to create request to GitHub: %w", err)
	}
	req.Header.Add("accept", "application/vnd.github.v3+json")
	req.Header.Add("accept", "application/vnd.github.groot-preview+json")
	req.Header.Add("authorization", fmt.Sprintf("token %s", g.token))

	httpResponse, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fail to execute the request to GitHub: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response code: %d", httpResponse.StatusCode)
	}

	payload, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("fail to read the response from GitHub: %w", err)
	}

	rawResponse := make([]map[string]interface{}, 0)
	if err := json.Unmarshal(payload, &rawResponse); err != nil {
		return nil, fmt.Errorf("fail to unmarshal the response from GitHub: %w", err)
	}

	ids := make([]int, 0, len(rawResponse))
	for _, entry := range rawResponse {
		id, ok := entry["number"].(float64)
		if !ok {
			return nil, errors.New("fail to unmarshal to cast the id into a integer")
		}
		ids = append(ids, (int)(id))
	}
	return ids, nil
}
