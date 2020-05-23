package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"nitro/markdown-link-check/internal/service/parser"
)

func TestFileInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		message   string
		client    File
		shouldErr bool
	}{
		{
			message:   "have an error because of it's missing the path",
			client:    File{Parser: &parser.Markdown{}},
			shouldErr: true,
		},
		{
			message:   "have an error because of it's missing the parser",
			client:    File{Path: "something"},
			shouldErr: true,
		},
		{
			message:   "succeed",
			client:    File{Path: "something", Parser: &parser.Markdown{}},
			shouldErr: false,
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

func TestFileAuthority(t *testing.T) {
	t.Parallel()

	client := File{Path: "something", Parser: &parser.Markdown{}}
	require.NoError(t, client.Init())

	tests := []struct {
		message      string
		uri          string
		hasAuthority bool
	}{
		{
			message:      "have authority #1",
			uri:          "/etc/hosts",
			hasAuthority: true,
		},
		{
			message:      "have authority #2",
			uri:          "actually this provider matches anything",
			hasAuthority: true,
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

func TestFileValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		message   string
		ctx       context.Context
		path      string
		uri       string
		isValid   bool
		shouldErr bool
		reader    func() *fileReaderMock
	}{
		{
			message:   "attest that the file doesn't exist",
			ctx:       context.Background(),
			path:      "file",
			uri:       "link",
			isValid:   false,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "link").Return(fileInfoMock{}, false)
				return &reader
			},
		},
		{
			message:   "attest that the file exists",
			ctx:       context.Background(),
			path:      "file",
			uri:       "link",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "link").Return(fileInfoMock{}, true)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file exists and the uri is valid #1",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "another-file",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "another-file").Return(fileInfoMock{}, true)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file exists and the uri is valid #2",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "another-file.md",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "another-file.md").Return(fileInfoMock{}, true)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file exists and the anchor is correct",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "#anchor",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "file.md").Return(fileInfoMock{}, true)
				reader.On("readFile", "file.md").Return([]byte("#anchor"), nil)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file exists and the uri is valid",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "another-file",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "another-file").Return(fileInfoMock{}, true)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file exists and the uri is directory",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "directory",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "directory").Return(fileInfoMock{isDirValue: true}, true)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file has the anchor #1",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "#anchor",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "file.md").Return(fileInfoMock{}, true)
				reader.On("readFile", "file.md").Return([]byte("# anchor"), nil)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file has the anchor #2",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "#anchor",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "file.md").Return(fileInfoMock{}, true)
				reader.On("readFile", "file.md").Return([]byte("# Anchor"), nil)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file has the anchor #3",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "another.md#anchor",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "another.md").Return(fileInfoMock{}, true)
				reader.On("readFile", "another.md").Return([]byte("# Anchor"), nil)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file has the anchor #4",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "another.md#Anchor",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "another.md").Return(fileInfoMock{}, true)
				reader.On("readFile", "another.md").Return([]byte("# anchor"), nil)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file has the anchor #5",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "another.md#Anchor",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				payload := bytes.NewBufferString("")
				fmt.Fprintln(payload, "# anchor")
				fmt.Fprintln(payload, "# another-one")

				var reader fileReaderMock
				reader.On("fileExists", "another.md").Return(fileInfoMock{}, true)
				reader.On("readFile", "another.md").Return(payload.Bytes(), nil)
				return &reader
			},
		},
		{
			message:   "attest that the markdown file has the anchor #6",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "another.md#anchor",
			isValid:   true,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "another.md").Return(fileInfoMock{}, true)
				reader.On("readFile", "another.md").Return([]byte("# [anchor](http://endpoint)"), nil)
				return &reader
			},
		},
		{
			message:   "attest that that an error happens during the file read",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "another-file.md#anchor",
			isValid:   false,
			shouldErr: true,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "another-file.md").Return(fileInfoMock{}, true)
				reader.On("readFile", "another-file.md").Return([]byte{}, errors.New("failed to read the file"))
				return &reader
			},
		},
		{
			message:   "attest that the markdown file path is invalid",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "http://192.168.0.%31",
			isValid:   false,
			shouldErr: true,
			reader:    func() *fileReaderMock { return &fileReaderMock{} },
		},
		{
			message:   "attest that the file doesn't exists",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "link.md",
			isValid:   false,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "link.md").Return(fileInfoMock{}, false)
				return &reader
			},
		},
		{
			message:   "attest that the file exists but the anchor doesn't #1",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "#anchor",
			isValid:   false,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "file.md").Return(fileInfoMock{}, true)
				reader.On("readFile", "file.md").Return([]byte("#another"), nil)
				return &reader
			},
		},
		{
			message:   "attest that the file exists but the anchor doesn't #2",
			ctx:       context.Background(),
			path:      "file.md",
			uri:       "#anchor",
			isValid:   false,
			shouldErr: false,
			reader: func() *fileReaderMock {
				var reader fileReaderMock
				reader.On("fileExists", "file.md").Return(fileInfoMock{}, true)
				reader.On("readFile", "file.md").Return([]byte(`<h5 key="value">value</h5>`), nil)
				return &reader
			},
		},
	}

	for i := 0; i < len(tests); i++ {
		tt := tests[i]
		t.Run("Should "+tt.message, func(t *testing.T) {
			t.Parallel()

			reader := tt.reader()
			defer reader.AssertExpectations(t)

			var parser parser.Markdown
			parser.Init()

			client := File{Path: "something", Parser: parser, reader: reader}
			require.NoError(t, client.Init())

			isValid, err := client.Valid(tt.ctx, tt.path, tt.uri)
			require.Equal(t, tt.shouldErr, (err != nil))
			if err != nil {
				return
			}
			require.Equal(t, tt.isValid, isValid)
		})
	}
}

type fileReaderMock struct {
	mock.Mock
}

func (f *fileReaderMock) fileExists(path string) (os.FileInfo, bool) {
	args := f.Called(path)
	return args.Get(0).(os.FileInfo), args.Bool(1)
}

func (f *fileReaderMock) readFile(path string) ([]byte, error) {
	args := f.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

type fileInfoMock struct {
	os.FileInfo
	isDirValue bool
}

func (fi fileInfoMock) IsDir() bool {
	return fi.isDirValue
}
