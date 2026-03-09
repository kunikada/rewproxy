package rule

import "net/http"

type HostRewriteRule struct{ From, To string }

func (r *HostRewriteRule) Apply(req *http.Request) error {
	if req.URL.Host == r.From {
		req.URL.Host = r.To
		req.Host = r.To
	}
	return nil
}
