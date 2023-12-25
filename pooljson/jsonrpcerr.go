package pooljson

// Standard JSON-RPC 2.0 errors.
var (
	ErrRPCInvalidRequest = &RPCError{
		Code:    -32600,
		Message: "Invalid request",
	}
	ErrRPCMethodNotFound = &RPCError{
		Code:    -32601,
		Message: "Method not found",
	}
	ErrRPCInvalidParams = &RPCError{
		Code:    -32602,
		Message: "Invalid parameters",
	}
	ErrRPCInternal = &RPCError{
		Code:    -32603,
		Message: "Internal error",
	}
	ErrRPCParse = &RPCError{
		Code:    -32700,
		Message: "Parse error",
	}
)

var (
	ErrInvalidUsername = &RPCError{
		Code:    200,
		Message: "Invalid username (1. letters, numbers and @_. are allowed. 2. length 6 to 100)",
	}
	ErrInvalidPassword = &RPCError{
		Code:    201,
		Message: "Invalid password (1. letters, numbers and ~!@#$%^&*?_- are allowed. 2. length 6 to 40)",
	}
	ErrPasswordIncorrect = &RPCError{
		Code:    202,
		Message: "Incorrect password",
	}
	ErrHelloNotReceived = &RPCError{
		Code:    300,
		Message: "Receive other requests before mining.hello",
	}
	ErrUnsubscribed = &RPCError{
		Code:    301,
		Message: "User unsubscribed",
	}
	ErrUnauthorized = &RPCError{
		Code:    302,
		Message: "User unauthorized",
	}
	ErrBadProtocolRequest = &RPCError{
		Code:    400,
		Message: "Bad protocol request",
	}
	ErrInvalidRequestParams = &RPCError{
		Code:    401,
		Message: "Invalid request params",
	}
	ErrAlreadyAuthorized = &RPCError{
		Code:    402,
		Message: "Already authorized",
	}
	ErrDuplicateShare = &RPCError{
		Code:    403,
		Message: "Duplicate share",
	}
	ErrJobNotFound = &RPCError{
		Code:    404,
		Message: "Job not found",
	}
	ErrInvalidShare = &RPCError{
		Code:    405,
		Message: "Invalid share",
	}
	ErrAlreadyRegistered = &RPCError{
		Code:    406,
		Message: "Username already registered",
	}
	ErrAddressInvalid = &RPCError{
		Code:    407,
		Message: "Address invalid",
	}
	ErrNotRegistered = &RPCError{
		Code:    408,
		Message: "Username not registered",
	}
	ErrNotFound = &RPCError{
		Code:    409,
		Message: "Not found",
	}
	ErrBadUserAgent = &RPCError{
		Code:    410,
		Message: "Bad user agent",
	}
	ErrInternal = &RPCError{
		Code:    500,
		Message: "Internal error",
	}
	ErrExceedMinerLimit = &RPCError{
		Code:    501,
		Message: "Miner limitation exceed, please try again later",
	}
)

func ErrAlreadyRegisteredWithUsername(username string) *RPCError {
	return &RPCError{
		Code:    406,
		Message: "Username " + username + " already registered",
	}
}

// General application defined JSON errors.
const (
	ErrRPCMisc                RPCErrorCode = -1
	ErrRPCForbiddenBySafeMode RPCErrorCode = -2
	ErrRPCType                RPCErrorCode = -3
	ErrRPCInvalidAddressOrKey RPCErrorCode = -5
	ErrRPCOutOfMemory         RPCErrorCode = -7
	ErrRPCInvalidParameter    RPCErrorCode = -8
	ErrRPCDatabase            RPCErrorCode = -20
	ErrRPCDeserialization     RPCErrorCode = -22
	ErrRPCVerify              RPCErrorCode = -25
	ErrRPCInWarmup            RPCErrorCode = -28
)

var (
	ErrUnknown = &RPCError{
		Code:    20,
		Message: "Error unknown",
	}
	ErrOutdatedTask = &RPCError{
		Code:    21,
		Message: "Outdated task",
	}
	ErrDuplicatedSubmit = &RPCError{
		Code:    22,
		Message: "Duplicated submit",
	}
	ErrUnSubscribed = &RPCError{
		Code:    24,
		Message: "Unsubscribe user",
	}
	ErrUnAuthorized = &RPCError{
		Code:    25,
		Message: "Unauthorized user",
	}
	ErrUsernameIncorrect = &RPCError{
		Code:    -102,
		Message: "Incorrect username",
	}
	ErrTimestampExpired = &RPCError{
		Code:    -103,
		Message: "Timestamp expired",
	}
	ErrAlreadyLogin = &RPCError{
		Code:    -104,
		Message: "Already login",
	}
	ErrUsernameNotFound = &RPCError{
		Code:    -200,
		Message: "Username not found",
	}
)

const (
	ErrRPCUnimplemented RPCErrorCode = -1
)
