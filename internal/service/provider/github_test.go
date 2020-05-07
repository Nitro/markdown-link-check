package provider

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/require"
)

func TestGitHubInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		message   string
		client    GitHub
		shouldErr bool
	}{
		{
			message:   "have an error because of it's missing the HTTP client",
			client:    GitHub{Token: "token", Owner: "owner"},
			shouldErr: true,
		},
		{
			message:   "have an error because of it's missing the token",
			client:    GitHub{HTTPClient: http.DefaultClient, Owner: "owner"},
			shouldErr: true,
		},
		{
			message:   "have an error because of it's missing the owner",
			client:    GitHub{HTTPClient: http.DefaultClient, Token: "token"},
			shouldErr: true,
		},
	}

	for i := 0; i < len(tests); i++ {
		tt := tests[i]
		t.Run("Should "+tt.message, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.shouldErr, (tt.client.Init() != nil))
		})
	}
}

func TestGitHubAuthority(t *testing.T) {
	t.Parallel()

	client := GitHub{HTTPClient: http.DefaultClient, Token: "token", Owner: "owner"}
	require.NoError(t, client.Init())

	tests := []struct {
		message      string
		uri          string
		hasAuthority bool
	}{
		{
			message:      "have authority #1",
			uri:          "https://github.com/owner",
			hasAuthority: true,
		},
		{
			message:      "have authority #2",
			uri:          "https://github.com/Owner",
			hasAuthority: true,
		},
		{
			message:      "have authority #3",
			uri:          "http://github.com/owner",
			hasAuthority: true,
		},
		{
			message:      "have authority #4",
			uri:          "http://github.com/Owner",
			hasAuthority: true,
		},
		{
			message:      "have authority #5",
			uri:          "https://raw.githubusercontent.com/owner",
			hasAuthority: true,
		},
		{
			message:      "have authority #6",
			uri:          "https://raw.githubusercontent.com/Owner",
			hasAuthority: true,
		},
		{
			message:      "have authority #7",
			uri:          "http://raw.githubusercontent.com/owner",
			hasAuthority: true,
		},
		{
			message:      "have authority #8",
			uri:          "http://raw.githubusercontent.com/Owner",
			hasAuthority: true,
		},
		{
			message:      "have no authority #1",
			uri:          "https://google.com",
			hasAuthority: false,
		},
		{
			message:      "have no authority #2",
			uri:          "https://github.com/another-owner",
			hasAuthority: false,
		},
	}

	for i := 0; i < len(tests); i++ {
		tt := tests[i]
		t.Run("Should "+tt.message, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.hasAuthority, client.Authority(tt.uri))
		})
	}
}

func TestGitHubValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		message    string
		ctx        context.Context
		repository githubRepositoryMock
		uri        string
		isValid    bool
		shouldErr  bool
	}{
		{
			message:    "attest the URI as a valid owner",
			ctx:        context.Background(),
			uri:        "https://github.com/owner",
			isValid:    true,
			shouldErr:  false,
			repository: githubRepositoryMock{},
		},
		{
			message:   "attest the URI as a valid issue",
			ctx:       context.Background(),
			uri:       "https://github.com/owner/repository/issues/1",
			isValid:   true,
			shouldErr: false,
			repository: githubRepositoryMock{
				repo:    "repository",
				issueID: 1,
			},
		},
		{
			message:   "attest the URI as a invalid issue comment",
			ctx:       context.Background(),
			uri:       "https://github.com/owner/repository/issues/1#issuecomment-3",
			isValid:   true,
			shouldErr: false,
			repository: githubRepositoryMock{
				repo:           "repository",
				issueID:        1,
				issueCommentID: 3,
			},
		},
		{
			message:   "attest the URI as a valid commit",
			ctx:       context.Background(),
			uri:       "https://github.com/owner/repository/commit/c09ea6d",
			isValid:   true,
			shouldErr: false,
			repository: githubRepositoryMock{
				repo: "repository",
				ref:  "c09ea6d",
			},
		},
		{
			message:   "attest the URI as a valid pull request",
			ctx:       context.Background(),
			uri:       "https://github.com/owner/repository/pull/3",
			isValid:   true,
			shouldErr: false,
			repository: githubRepositoryMock{
				repo:          "repository",
				pullRequestID: 3,
			},
		},
		{
			message:   "attest the URI as a valid commit associated with the pull request",
			ctx:       context.Background(),
			uri:       "https://github.com/owner/repository/pull/3/commits/c09ea6d",
			isValid:   true,
			shouldErr: false,
			repository: githubRepositoryMock{
				repo:          "repository",
				pullRequestID: 3,
				ref:           "c09ea6d",
			},
		},
		{
			message:   "attest the URI as a valid repository",
			ctx:       context.Background(),
			uri:       "https://github.com/owner/repository",
			isValid:   true,
			shouldErr: false,
			repository: githubRepositoryMock{
				repo: "repository",
			},
		},
		{
			message:    "attest the URI as a invalid because of a invalid owner",
			ctx:        context.Background(),
			uri:        "https://github.com/another-owner/repository/issues/1",
			isValid:    false,
			shouldErr:  false,
			repository: githubRepositoryMock{},
		},
		{
			message:    "attest the URI as a invalid because of a invalid URI",
			ctx:        context.Background(),
			uri:        "https://github.com/owner/action",
			isValid:    false,
			shouldErr:  false,
			repository: githubRepositoryMock{},
		},
	}

	for i := 0; i < len(tests); i++ {
		tt := tests[i]
		t.Run("Should "+tt.message, func(t *testing.T) {
			t.Parallel()

			client := GitHub{HTTPClient: http.DefaultClient, Token: "token", Owner: "owner"}
			client.repository = tt.repository
			require.NoError(t, client.Init())

			isValid, err := client.Valid(tt.ctx, "", tt.uri)
			require.Equal(t, tt.shouldErr, (err != nil))
			if err != nil {
				return
			}
			require.Equal(t, tt.isValid, isValid)
		})
	}
}

type githubRepositoryMock struct {
	issueID        int
	issueCommentID int64
	repo           string
	ref            string
	pullRequestID  int
}

func (g githubRepositoryMock) repository(ctx context.Context, repository string) (*github.Response, error) {
	if g.repo != repository {
		return nil, errors.New("fail")
	}
	return &github.Response{Response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
	}}, nil
}

func (g githubRepositoryMock) repositoriesGetCommitSHA1(
	ctx context.Context, repository, ref string,
) (*github.Response, error) {
	if (g.repo != repository) || (g.ref != ref) {
		return nil, errors.New("fail")
	}
	return &github.Response{Response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
	}}, nil
}

func (g githubRepositoryMock) issuesGet(ctx context.Context, repo string, number int) (*github.Response, error) {
	if (g.repo != repo) || (g.issueID != number) {
		return nil, errors.New("fail")
	}
	return &github.Response{Response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
	}}, nil
}

func (g githubRepositoryMock) issuesGetComment(ctx context.Context, repo string, id int64) (*github.Response, error) {
	if (g.repo != repo) || (g.issueCommentID != id) {
		return nil, errors.New("fail")
	}
	return &github.Response{Response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
	}}, nil
}

func (g githubRepositoryMock) pullRequestsGetRaw(
	ctx context.Context, repo string, number int,
) (*github.Response, error) {
	if (g.repo != repo) || (g.pullRequestID != number) {
		return nil, errors.New("fail")
	}
	return &github.Response{Response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
	}}, nil
}

func (g githubRepositoryMock) relatedPullRequests(ctx context.Context, repository, ref string) ([]int, error) {
	if (g.repo != repository) || (g.ref != ref) {
		return nil, errors.New("fail")
	}
	return []int{g.pullRequestID}, nil
}
