package surf

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// JSONResp write content as JSON encoded response.
func JSONResp(code int, content interface{}) Response {
	b, err := json.MarshalIndent(content, "", "\t")
	if err != nil {
		return &jsonResponse{
			code: http.StatusInternalServerError,
			body: strings.NewReader(`{"errors":["Internal Server Errror"]}`),
		}
	}
	return &jsonResponse{
		code: code,
		body: bytes.NewReader(b),
	}
}

type jsonResponse struct {
	code int
	body io.Reader
}

func (resp *jsonResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(resp.code)
	io.Copy(w, resp.body)
}

// JSONErr write single error as JSON encoded response.
func JSONErr(code int, errText string) {
	JSONErrs(code, []string{errText})
}

// JSONErrs write multiple errors as JSON encoded response.
func JSONErrs(code int, errs []string) {
	resp := struct {
		Code   int      `json:"code"`
		Errors []string `json:"errors"`
	}{
		Code:   code,
		Errors: errs,
	}
	JSONResp(code, resp)
}

// StdJSONResp write JSON encoded, standard HTTP response text for given status
// code. Depending on status, either error or successful response format is
// used.
func StdJSONResp(code int) {
	if code >= 400 {
		JSONErr(code, http.StatusText(code))
	} else {
		JSONResp(code, http.StatusText(code))
	}
}
