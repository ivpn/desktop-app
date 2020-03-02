package types

type AuthenticationRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthenticationResponse struct {
	Status        int  `json:"status"`
	Authenticated bool `json:"authenticated"`
}
