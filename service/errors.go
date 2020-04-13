package service

// ErrorNotLoggedIn - error, user not logged in into account
type ErrorNotLoggedIn struct {
}

func (e ErrorNotLoggedIn) Error() string {
	return "not logged in; please visit https://www.ivpn.net/ to Sign Up or Log In to get info about your Account ID"
}
