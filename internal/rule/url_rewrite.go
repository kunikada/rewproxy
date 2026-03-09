package rule

import (
	"net/http"
	"strings"
)

type URLRewriteRule struct{ From, To string }

func (r *URLRewriteRule) Apply(req *http.Request) error {
	if strings.HasPrefix(req.URL.Path, r.From) {
		req.URL.Path = r.To + req.URL.Path[len(r.From):]
	}
	return nil
}
