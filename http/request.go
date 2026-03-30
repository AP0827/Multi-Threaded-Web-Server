package mtwshttp

type Request struct {
	Method  string
	path    string
	version string
	headers map[string]string
	body    []byte
}
