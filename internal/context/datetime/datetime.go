package datetime

import "time"

// Provider generates a formatted date header.
type Provider struct {
	now func() time.Time // injectable for testing
}

// New creates a new datetime content provider.
func New() *Provider {
	return &Provider{now: time.Now}
}

func (p *Provider) Name() string { return "datetime" }

func (p *Provider) Generate() (string, error) {
	t := p.now()
	return "# " + t.Format("Monday, January 2, 2006"), nil
}
