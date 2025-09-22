package webserver

type CTXClientIP struct{}
type CTXURLRequest struct{}
type CTXFileRequest struct {
	FullPath     string
	ResolvedPath string
	FileName     string
}

type HTTPErrorCodes struct {
	Message string `json:"message"`
}
