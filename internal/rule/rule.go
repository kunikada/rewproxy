package rule

import "net/http"

// Rule is the core interface every rewrite rule must satisfy.
// Apply mutates the outbound request. A non-nil error aborts the pipeline.
type Rule interface {
	Apply(req *http.Request) error
}

// Pipeline is an ordered slice of Rules applied left to right.
type Pipeline []Rule

func (p Pipeline) Apply(req *http.Request) error {
	for _, r := range p {
		if err := r.Apply(req); err != nil {
			return err
		}
	}
	return nil
}
