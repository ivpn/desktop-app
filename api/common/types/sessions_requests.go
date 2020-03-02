package types

// SessionAuthenticateRequest Sessions Authenticate request
type SessionAuthenticateRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	PublicKey  string `json:"wg_public_key"`
	ForceLogin bool   `json:"force"`
}

// SessionTokenRequest Sessions Status request
type SessionTokenRequest struct {
	Token string `json:"session_token"`
}

type SessionWireGuardResponse struct {
	Status    int    `json:"status"`
	IPAddress string `json:"ip_address"`
}

type SessionWireGuardAddKeyRequest struct {
	Token              string `json:"session_token"`
	PublicKey          string `json:"public_key"`
	ConnectedPublicKey string `json:"connected_public_key"`
}

type SessionWireGuardUpgradeKeyRequest struct {
	Token        string `json:"session_token"`
	PublicKey    string `json:"public_key"`
	NewPublicKey string `json:"new_public_key"`
}

type SessionWireGuardKeyRequest struct {
	Token     string `json:"session_token"`
	PublicKey string `json:"public_key"`
}
