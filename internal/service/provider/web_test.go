package provider

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebInit(t *testing.T) {
	t.Parallel()
	var client Web
	require.NoError(t, client.Init())
}

func TestWebAuthority(t *testing.T) {
	t.Parallel()

	var client Web
	require.NoError(t, client.Init())

	tests := []struct {
		message      string
		uri          string
		hasAuthority bool
	}{
		{
			message:      "have authority #1",
			uri:          "https://website.com",
			hasAuthority: true,
		},
		{
			message:      "have authority #2",
			uri:          "http://website.com",
			hasAuthority: true,
		},
		{
			message:      "have no authority #1",
			uri:          "../file.md",
			hasAuthority: false,
		},
		{
			message:      "have no authority #2",
			uri:          "/folder",
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

func TestWebValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		message   string
		ctx       context.Context
		client    webClientMock
		uri       string
		isValid   bool
		shouldErr bool
	}{
		{
			message:   "attest the URI as valid",
			ctx:       context.Background(),
			uri:       "https://go.dev",
			shouldErr: false,
			isValid:   true,
			client: webClientMock{
				request: []http.Request{
					{URL: &url.URL{Scheme: "https", Host: "go.dev"}},
				},
				response: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
					},
				},
			},
		},
		{
			message:   "attest the URI as valid after a move permanently redirect (301)",
			ctx:       context.Background(),
			uri:       "http://go.dev",
			shouldErr: false,
			isValid:   true,
			client: webClientMock{
				request: []http.Request{
					{URL: &url.URL{Scheme: "http", Host: "go.dev"}},
					{URL: &url.URL{Scheme: "https", Host: "go.dev"}},
				},
				response: []*http.Response{
					{
						Header:     map[string][]string{"Location": {"https://go.dev"}},
						StatusCode: http.StatusMovedPermanently,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
					},
					{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
					},
				},
			},
		},
		{
			message:   "attest the URI as valid after a permanent redirect (308)",
			ctx:       context.Background(),
			uri:       "http://go.dev",
			shouldErr: false,
			isValid:   true,
			client: webClientMock{
				request: []http.Request{
					{URL: &url.URL{Scheme: "http", Host: "go.dev"}},
					{URL: &url.URL{Scheme: "https", Host: "go.dev"}},
				},
				response: []*http.Response{
					{
						Header:     map[string][]string{"Location": {"https://go.dev"}},
						StatusCode: http.StatusPermanentRedirect,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
					},
					{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
					},
				},
			},
		},
		{
			message:   "attest the URI as valid and also have a valid anchor",
			ctx:       context.Background(),
			uri:       "https://go.dev#title",
			shouldErr: false,
			isValid:   true,
			client: webClientMock{
				request: []http.Request{
					{URL: &url.URL{Scheme: "https", Host: "go.dev", Fragment: "title"}},
				},
				response: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewBufferString(`<a href="#title"/>`)),
					},
				},
			},
		},
		{
			message:   "attest the URI as invalid because of a temporary redirect",
			ctx:       context.Background(),
			uri:       "http://go.dev",
			shouldErr: false,
			isValid:   false,
			client: webClientMock{
				request: []http.Request{
					{URL: &url.URL{Scheme: "https", Host: "go.dev"}},
				},
				response: []*http.Response{
					{
						Header:     map[string][]string{"Location": {"https://go.dev"}},
						StatusCode: http.StatusTemporaryRedirect,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
					},
				},
			},
		},
		{
			message:   "attest the URI as invalid because of a not found status",
			ctx:       context.Background(),
			uri:       "https://go.dev",
			shouldErr: false,
			isValid:   false,
			client: webClientMock{
				request: []http.Request{
					{URL: &url.URL{Scheme: "https", Host: "go.dev"}},
				},
				response: []*http.Response{
					{
						StatusCode: http.StatusNotFound,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
					},
				},
			},
		},
		{
			message:   "attest the URI as invalid because of a not found anchor",
			ctx:       context.Background(),
			uri:       "https://go.dev#broken",
			shouldErr: false,
			isValid:   false,
			client: webClientMock{
				request: []http.Request{
					{URL: &url.URL{Scheme: "https", Host: "go.dev", Fragment: "broken"}},
				},
				response: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewBufferString(`<a href="#title"/>`)),
					},
				},
			},
		},
	}

	for i := 0; i < len(tests); i++ {
		tt := tests[i]
		t.Run("Should "+tt.message, func(t *testing.T) {
			client := Web{client: &tt.client}
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

type webClientMock struct {
	request  []http.Request
	response []*http.Response
	index    int
}

func (c *webClientMock) Do(req *http.Request) (*http.Response, error) {
	var (
		request  = c.request[c.index]
		response = c.response[c.index]
	)
	c.index++
	if request.URL.String() != req.URL.String() {
		return nil, errors.New("invalid endpoint")
	}
	return response, nil
}
