package types

// SessionNewRequest request to create new session
type SessionNewRequest struct {
	AccountID  string `json:"username"`
	PublicKey  string `json:"wg_public_key"`
	ForceLogin bool   `json:"force"`
}

// SessionDeleteRequest request to delete session
type SessionDeleteRequest struct {
	Session string `json:"session_token"`
}

// SessionWireGuardKeySetRequest request to set new WK key for a session
type SessionWireGuardKeySetRequest struct {
	Session            string `json:"session_token"`
	PublicKey          string `json:"public_key"`
	ConnectedPublicKey string `json:"connected_public_key"`
}
