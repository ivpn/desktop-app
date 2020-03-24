package commands

// NotImplemented error
type NotImplemented struct {
	Message string
}

func (e NotImplemented) Error() string {
	if len(e.Message) == 0 {
		return "not implemented"
	}
	return e.Message
}
