package service

// ErrorNotLoggedIn - error, user not logged in into account
type ErrorNotLoggedIn struct {
}

func (e ErrorNotLoggedIn) Error() string {
	return "not logged in"
}
