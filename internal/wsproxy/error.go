package wsproxy

type backendClosedError struct{}

func (err backendClosedError) Error() string {
	return "connection to backend is closed"
}

type clientError struct {
	err error
}

func (err clientError) Error() string {
	return err.err.Error()
}
