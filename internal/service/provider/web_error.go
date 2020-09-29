package provider

import (
	"net/http"

	"github.com/kr/pretty"
)

type webError struct {
	base           error
	requestHeader  http.Header
	responseHeader http.Header
	status         int
	body           string
}

func (err webError) Error() string {
	return err.base.Error()
}

func (err webError) PrettyPrint() {
	pretty.Println(err)
}
