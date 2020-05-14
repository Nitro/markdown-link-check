package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmailInit(t *testing.T) {
	t.Parallel()
	var client Email
	require.NoError(t, client.Init())
}

func TestEmailAuthority(t *testing.T) {
	t.Parallel()

	var client Email
	require.NoError(t, client.Init())

	tests := []struct {
		message      string
		uri          string
		hasAuthority bool
	}{
		{
			message:      "have authority #1",
			uri:          "mailto:milo@gonitro.com",
			hasAuthority: true,
		},
		{
			message:      "have authority #2",
			uri:          "mailto:milo@gonitro.com?subject=something",
			hasAuthority: true,
		},
		{
			message:      "have no authority #1",
			uri:          "http://something.com",
			hasAuthority: false,
		},
		{
			message:      "have no authority #2",
			uri:          "milo@gonitro.com",
			hasAuthority: false,
		},
		{
			message:      "have no authority #2",
			uri:          "milo@gonitro.com?subject=something",
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

func TestEmailValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		message   string
		ctx       context.Context
		checker   emailChecker
		uri       string
		isValid   bool
		shouldErr bool
	}{
		{
			message:   "attest the email as valid",
			ctx:       context.Background(),
			checker:   emailCheckerMock{response: true, shouldErr: false},
			uri:       "mailto:milo@gonitro.com",
			shouldErr: false,
			isValid:   true,
		},
		{
			message:   "attest the email with name as valid",
			ctx:       context.Background(),
			checker:   emailCheckerMock{response: true, shouldErr: false},
			uri:       "mailto:milo@gonitro.com?subject=something",
			shouldErr: false,
			isValid:   true,
		},
		{
			message:   "attest the email as invalid #1",
			ctx:       context.Background(),
			checker:   emailCheckerMock{response: true, shouldErr: false},
			uri:       "something",
			shouldErr: false,
			isValid:   false,
		},
		{
			message:   "attest the email as invalid #2",
			ctx:       context.Background(),
			checker:   emailCheckerMock{response: true, shouldErr: false},
			uri:       "http://gonitro.com",
			shouldErr: false,
			isValid:   false,
		},
		{
			message:   "attest the email as invalid #3",
			ctx:       context.Background(),
			checker:   emailCheckerMock{response: false, shouldErr: false},
			uri:       "unknow@email.com",
			shouldErr: false,
			isValid:   false,
		},
		{
			message:   "attest the email as invalid #4",
			ctx:       context.Background(),
			checker:   emailCheckerMock{response: false, shouldErr: true},
			uri:       "mailto:valid@email.com",
			shouldErr: true,
			isValid:   false,
		},
	}

	for i := 0; i < len(tests); i++ {
		tt := tests[i]
		t.Run("Should "+tt.message, func(t *testing.T) {
			client := Email{checker: tt.checker}
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

type emailCheckerMock struct {
	response  bool
	shouldErr bool
}

func (e emailCheckerMock) exists(domain string) (bool, error) {
	if e.shouldErr {
		return false, errors.New("error during the email check")
	}
	return e.response, nil
}
