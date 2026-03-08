package http_proxy_plugin

// ResultKind represents plugin execution outcome.
type ResultKind int

const (
	ResultContinue ResultKind = iota
	ResultAbort
)

// Result is the standardized plugin execution result.
type Result struct {
	Kind       ResultKind
	Err        error
	HTTPStatus int
	Code       int
	Message    string
}

func Continue() Result {
	return Result{Kind: ResultContinue}
}

func Abort(err error) Result {
	return Result{Kind: ResultAbort, Err: err}
}

func AbortWithStatus(httpStatus int, err error) Result {
	return Result{Kind: ResultAbort, Err: err, HTTPStatus: httpStatus}
}

func AbortWithCode(httpStatus, code int, message string, err error) Result {
	return Result{Kind: ResultAbort, Err: err, HTTPStatus: httpStatus, Code: code, Message: message}
}

func (r Result) IsAbort() bool {
	return r.Kind == ResultAbort
}
