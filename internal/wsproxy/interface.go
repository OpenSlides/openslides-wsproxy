package wsproxy

// GetURLer returns a full url for a url path.
type GetURLer interface {
	GetURL(url string) string
}

type command interface {
	Call(conn *wsConnection) error
}
