package cmdline

type parseError struct {
	message string
}

func (e *parseError) Error() string {
	return e.message
}

type ValueHandler interface {
	Notify(text string, log Logger) bool
	Complete(text string, observer CompletionObserver)
	TypeName() string
}
