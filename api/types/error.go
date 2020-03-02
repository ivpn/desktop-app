package types

// APIError generic IVPN API error
type APIError struct {
	Status  int    `json:"status"`  // ID of the error, so Clients can avoid parsing text output.
	Message string `json:"message"` // Text description of the message
}
