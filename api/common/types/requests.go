package types

// APIErrorResponse is the structure returned
// for json request when error happens
type APIErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type GeoLookupRequest struct {
	IPAddress string `json:"ip_address"`
}

type WireGuardResponse struct {
	Status    int    `json:"status"`
	IPAddress string `json:"ip_address"`
}

type WireGuardAddKeyRequest struct {
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
	Comment   string `json:"comment"`
	IsSystem  bool   `json:"is_system"`
}

type WireGuardUpgradeKeyRequest struct {
	Username     string `json:"username"`
	PublicKey    string `json:"public_key"`
	NewPublicKey string `json:"new_public_key"`
}

type WireGuardKeyRequest struct {
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
}
