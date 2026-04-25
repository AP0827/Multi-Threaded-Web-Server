package mtwshttp

type Request struct {
	method  string
	path    string
	version string
	headers map[string]string
	body    []byte
}

func (r *Request) Method() string {
	return r.method
}

func (r *Request) Path() string {
	return r.path
}

func (r *Request) Version() string {
	return r.version
}

func (r *Request) Headers() map[string]string {
	return r.headers
}

func (r *Request) Body() string {
	return string(r.body)
}
