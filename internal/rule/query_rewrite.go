package rule

import "net/http"

type QueryRewriteRule struct{ Name, Value string }

func (r *QueryRewriteRule) Apply(req *http.Request) error {
	q := req.URL.Query()
	q.Set(r.Name, r.Value)
	req.URL.RawQuery = q.Encode()
	return nil
}
