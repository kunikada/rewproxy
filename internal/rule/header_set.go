package rule

import "net/http"

type HeaderSetRule struct{ Name, Value string }

func (r *HeaderSetRule) Apply(req *http.Request) error {
	req.Header.Set(r.Name, r.Value)
	return nil
}
