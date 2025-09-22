package webserver

type ctxClientIP struct{}
type ctxURLRequest struct{}
type ctxFileRequest struct {
	FullPath     string
	ResolvedPath string
	FileName     string
}

type httpErrorCodes struct {
	Message string `json:"message"`
}
