package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

	// The fragment is not sent at the HTTP request, so we have some weird endpoints like
	// '/valid-fragment-title-from-browser' to have the needed granularity to execute the tests.
	//
	// More details on this issue: https://github.com/golang/go/issues/3805#issuecomment-66068331
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "true", r.Header.Get("control"))

		const validEndpoint = "/valid"

		if r.URL.Path == validEndpoint {
			return
		}

		if r.URL.Path == "/301" {
			r.URL.Path = validEndpoint
			w.Header().Set("location", r.URL.String())
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}

		if r.URL.Path == "/308" {
			r.URL.Path = validEndpoint
			w.Header().Set("location", r.URL.String())
			w.WriteHeader(http.StatusPermanentRedirect)
			return
		}

		if r.URL.Path == "/valid-fragment-title" {
			_, err := w.Write([]byte(`<a href="#title"/>`))
			require.NoError(t, err)
			return
		}

		if r.URL.Path == "/valid-fragment-title-from-browser" {
			var response string
			if r.Header.Get("user-agent") != "Go-http-client/1.1" {
				response = `<a href="#title"/>`
			}
			_, err := w.Write([]byte(response))
			require.NoError(t, err)
			return
		}

		if r.URL.Path == "/valid-user-agent-chrome" {
			if r.Header.Get("user-agent") == "chrome" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		if r.URL.Path == "/valid-user-agent-firefox" {
			if r.Header.Get("user-agent") == "firefox" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		if r.URL.Path == "/307" {
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}

		if r.URL.Path == "/404" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/invalid-fragment-broken" {
			require.Equal(t, "true", r.Header.Get("control-browser"))
			_, err := w.Write([]byte(`<a href="#title"/>`))
			require.NoError(t, err)
			return
		}

		require.FailNow(t, "not expected to reach this point")
	}))
	defer server.Close()
	serverEndpoint, err := url.Parse(server.URL)
	require.NoError(t, err)

	tests := []struct {
		message   string
		endpoint  url.URL
		isValid   bool
		shouldErr bool
	}{
		{
			message:   "attest the URI as valid",
			endpoint:  url.URL{Path: "/valid"},
			shouldErr: false,
			isValid:   true,
		},
		{
			message:   "attest the URI as valid after a move permanently redirect (301)",
			endpoint:  url.URL{Path: "/301"},
			shouldErr: false,
			isValid:   true,
		},
		{
			message:   "attest the URI as valid after a permanent redirect (308)",
			endpoint:  url.URL{Path: "/308"},
			shouldErr: false,
			isValid:   true,
		},
		{
			message:   "attest the URI as valid and also have a valid anchor",
			endpoint:  url.URL{Path: "/valid-fragment-title", Fragment: "title"},
			shouldErr: false,
			isValid:   true,
		},
		{
			message:   "attest the URI as valid and also have a valid anchor at the browser",
			endpoint:  url.URL{Path: "/valid-fragment-title-from-browser", Fragment: "title"},
			shouldErr: false,
			isValid:   true,
		},
		{
			message:   "attest the URI as valid and have the correct user agent #1",
			endpoint:  url.URL{Path: "/valid-user-agent-chrome", Opaque: "chrome"},
			shouldErr: false,
			isValid:   true,
		},
		{
			message:   "attest the URI as valid and have the correct user agent #2",
			endpoint:  url.URL{Path: "/valid-user-agent-firefox"},
			shouldErr: false,
			isValid:   true,
		},
		{
			message:   "attest the URI as invalid because of a temporary redirect",
			endpoint:  url.URL{Path: "/307"},
			shouldErr: false,
			isValid:   false,
		},
		{
			message:   "attest the URI as invalid because of a not found status",
			endpoint:  url.URL{Path: "/404"},
			shouldErr: false,
			isValid:   false,
		},
		{
			message:   "attest the URI as invalid because of a not found anchor",
			endpoint:  url.URL{Path: "/invalid-fragment-broken", Fragment: "broken"},
			shouldErr: false,
			isValid:   false,
		},
	}

	genEndpoint := func(endpoint url.URL) string {
		result := *serverEndpoint
		result.Path = endpoint.Path
		result.Fragment = endpoint.Fragment
		if endpoint.Opaque == "chrome" {
			result.Host = strings.ReplaceAll(result.Host, "127.0.0.1", "localhost")
		}
		return result.String()
	}

	for i := 0; i < len(tests); i++ {
		tt := tests[i]
		t.Run("Should "+tt.message, func(t *testing.T) {
			client := Web{
				Config: WebConfig{Header: make(http.Header)},
				ConfigOverwrite: map[string]WebConfig{
					"http://localhost": {Header: make(http.Header)},
					"http://127.0.0.1": {Header: make(http.Header)},
				},
			}
			client.Config.Header.Set("control", "true")
			client.Config.Header.Set("user-agent", "firefox")
			client.ConfigOverwrite["http://localhost"].Header.Set("user-agent", "chrome")
			client.ConfigOverwrite["http://127.0.0.1"].Header.Set("control-browser", "true")
			require.NoError(t, client.Init())
			defer client.Close()

			isValid, err := client.Valid(context.Background(), "", genEndpoint(tt.endpoint))
			require.Equal(t, tt.shouldErr, (err != nil))
			if err != nil {
				return
			}
			require.Equal(t, tt.isValid, isValid)
		})
	}
}
