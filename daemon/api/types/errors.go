//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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

const (
	// CodeSuccess - success
	CodeSuccess int = 200

	// Unauthorized - Invalid Credentials	(Username or Password is not valid)
	Unauthorized int = 401

	// WGPublicKeyNotFound - WireGuard Public Key not found
	WGPublicKeyNotFound int = 424

	// SessionNotFound - Session not found Session not found
	SessionNotFound int = 601

	// CodeSessionsLimitReached - You've reached the session limit, log out from other device
	CodeSessionsLimitReached int = 602

	// AccountNotActive - account should be purchased
	AccountNotActive int = 702

	CaptchaRequired int = 70001
	CaptchaInvalid  int = 70002

	// Account has two-factor authentication enabled. Please enter TOTP token to login
	The2FARequired int = 70011
	// Specified two-factor authentication token is not valid
	The2FAInvalidToken int = 70012
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
