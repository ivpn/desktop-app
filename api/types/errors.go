//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package types

import "fmt"

// APIErrorCode API error code
type APIErrorCode int

const (
	// CodeSuccess - success
	CodeSuccess APIErrorCode = 200

	// Unauthorized - Invalid Credentials	(Username or Password is not valid)
	Unauthorized APIErrorCode = 401

	// WGPublicKeyNotFound - WireGuard Public Key not found
	WGPublicKeyNotFound APIErrorCode = 424

	// SessionNotFound - Session not found Session not found
	SessionNotFound APIErrorCode = 601

	// CodeSessionsLimitReached - You've reached the session limit, log out from other device
	CodeSessionsLimitReached APIErrorCode = 602

	// AccountNotActive - account should be purchased
	AccountNotActive APIErrorCode = 702
)

// APIError - error, user not logged in into account
type APIError struct {
	ErrorCode int
	Message   string
}

// CreateAPIError creates new API error object
func CreateAPIError(errorCode int, message string) APIError {
	return APIError{
		ErrorCode: errorCode,
		Message:   message}
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error: [%d] %s", e.ErrorCode, e.Message)
}
