package service

// Entry represents the link present at a given file.
type Entry struct {
	Path       string
	Link       string
	Valid      bool
	FailReason func()
}

// EnhancedError implements a logic to pretty print an error.
type EnhancedError interface {
	PrettyPrint()
}
