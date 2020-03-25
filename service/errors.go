package service

// ErrorNotLoggedIn - error, user not logged in into account
type ErrorNotLoggedIn struct {
}

func (e ErrorNotLoggedIn) Error() string {
	return "not logged in; please, login (you can visit https://www.ivpn.net/ to Sing Up or to get info about your account ID)"
}
