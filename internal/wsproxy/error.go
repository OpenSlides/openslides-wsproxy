package wsproxy

type backendClosedError struct{}

func (err backendClosedError) Error() string {
	return "connection to backend is closed"
}
