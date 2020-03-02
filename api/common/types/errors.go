package types

const CodeSuccess = 200

const CodeBadRequest = 400
const CodeBadRequestJSONMessage = "No or invalid JSON detected in request body"

const CodeServerError = 500
const CodeServerErrorMessage = "Unknown error occurred"

const CodeInvalidCredentials = 401
const CodeInvalidCredentialsMessage = "Username or Password is not valid"

const CodeWireGuardKeyInvalidUsername = 421
const CodeWireGuardKeyInvalidUsernameMessage = "Cannot add WireGuard Public Key for service without username"
const CodeWireGuardKeyNotValid = 422
const CodeWireGuardKeyNotValidMessage = "Public key is not valid. Key should be exactly 32 bytes base64 encoded."
const CodeWireGuardKeyAlreadyExists = 423
const CodeWireGuardKeyAlreadyExistsMessage = "Specified public key already exists."
const CodeWireGuardKeyNotFound = 424
const CodeWireGuardKeyNotFoundMessage = "Public Key not found"
const CodeWireGuardKeyLimitReached = 425
const CodeWireGuardKeyLimitReachedMessage = "WireGuard key limit is reached"
const CodeWireGuardKeyNotProvided = 426
const CodeWireGuardKeyNotProvidedMessage = "WireGuard public key was not provided"

const CodeGeoLookupDBError = 501
const CodeGeoLookupDBErrorMessage = "Error while connecting to GEO IP database"
const CodeGeoLookupIPInvalid = 502
const CodeGeoLookupIPInvalidMessage = "Invalid IP"
const CodeGeoLookupIPNotFound = 503
const CodeGeoLookupIPNotFoundMessage = "Error whilst finding city based on IP"

const CodeServiceNotFoundRequest = 600
const CodeServiceNotFoundMessage = "Service not found"
const CodeServiceDeletedRequest = 601
const CodeServiceDeletedMessage = "Service is deleted"
const CodeServiceNotActiveRequest = 602
const CodeServiceNotActiveMessage = "Service is not active"
