package provider

import (
	"context"
	"errors"
	"io"
	"nitro/markdown-link-check/internal/service/parser"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks
// Mock: Simple File
type MockFileSimple struct {
	mock.Mock
}

func (f MockFileSimple) regexCompile(str string) (*regexp.Regexp, error) {
	args := f.Called(str)
	return nil, args.Error(1)
}
func (f MockFileSimple) fileExists(str string) (os.FileInfo, bool) {
	args := f.Called(str)
	finfo := fileInfo{}
	return finfo, args.Bool(0)
}
func (f MockFileSimple) readFile(str string) ([]byte, error) {
	args := f.Called(str)
	fakecontent := []byte(args.String((0)))
	return fakecontent, nil
}

func (f MockFileSimple) docQuery(r io.Reader) (*goquery.Document, error) {
	args := f.Called(r)
	return nil, args.Error(1)
}

type fileInfo struct {
}

func (m fileInfo) Name() string {
	return ""
}

func (m fileInfo) Size() int64 {
	return 0
}

func (m fileInfo) Mode() os.FileMode {
	return os.FileMode(0)
}

func (m fileInfo) ModTime() time.Time {
	return time.Time{}
}

func (m fileInfo) IsDir() bool {
	return false
}

func (m fileInfo) Sys() interface{} {
	return nil
}

// Test: Init
func TestInit(t *testing.T) {
	// Fail: No Path
	f := File{Path: ""}
	err := f.Init()
	assert.EqualError(t, err, "missing 'path'")

	// Fail: No Parser
	h := FileHelpersC{}

	f = File{Path: "fakepath", Parser: nil, Helpers: h}
	err = f.Init()
	assert.EqualError(t, err, "missing 'parser'")

	// Fail: regexCompile to fail
	var p parser.Markdown
	mh := new(MockFileSimple)

	mh.On("regexCompile", "^.*$").Return(nil, errors.New("Force Regex Compile to fail"))

	f = File{Path: "fakepath", Parser: p, Helpers: mh}
	err = f.Init()
	assert.EqualError(t, err, "fail to initialize the regex expressions: fail to compile the expression '^.*$': Force Regex Compile to fail")
}

/** Complete Test Cases **/
// Test case: Valid link (existent file)
func TestValid(t *testing.T) {
	var p parser.Markdown
	l := "./fake_file"
	pt := "."

	ctx, _ := context.WithCancel(context.Background())
	mh := new(MockFileSimple)
	mh.On("fileExists", "fake_file").Return(true)

	f := File{Path: pt, Parser: p, Helpers: mh}

	valid, _ := f.Valid(ctx, pt, l)
	assert.True(t, valid)
}

// Test case: Invalid link (unexistent file)
func TestInvalid(t *testing.T) {
	var p parser.Markdown
	l := "./fake_file"
	pt := "."

	ctx, _ := context.WithCancel(context.Background())
	mh := new(MockFileSimple)
	mh.On("fileExists", "fake_file").Return(false)

	f := File{Path: pt, Parser: p, Helpers: mh}

	valid, _ := f.Valid(ctx, pt, l)
	assert.False(t, valid)
}
