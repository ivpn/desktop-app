package flags

// BadParameter error
type BadParameter struct {
	Message string
}

func (e BadParameter) Error() string {
	if len(e.Message) == 0 {
		return "bad parameter"
	}
	return "bad parameter (" + e.Message + ")"
}
