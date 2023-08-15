//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
//
//  This file is part of the UI for IVPN Client Desktop.
//
//  The UI for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The UI for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the UI for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

export const API_SUCCESS = 200;
export const API_SESSION_LIMIT = 602;

export const API_CAPTCHA_REQUIRED = 70001;
export const API_CAPTCHA_INVALID = 70002;

export const API_2FA_REQUIRED = 70011; // Account has two-factor authentication enabled. Please enter TOTP token to login
export const API_2FA_TOKEN_NOT_VALID = 70012; // Specified two-factor authentication token is not valid
