package xerror

type ErrInvalidUsername struct{}

func (e *ErrInvalidUsername) Error() string {
	return "error code: 001 - message: invalid username"
}

type ErrInvalidProtocol struct{}

func (e *ErrInvalidProtocol) Error() string {
	return "error code: 002 - message: invalid protocol"
}

type ErrInvalidPortNumberCode1 struct{}

func (e *ErrInvalidPortNumberCode1) Error() string {
	return "error code: 003 - message: invalid port number"
}

type ErrInvalidPortNumberCode2 struct{}

func (e *ErrInvalidPortNumberCode2) Error() string {
	return "error code: 004 - message: invalid port number"
}

type ErrJWTAccessTokenNotFound struct{}

func (e *ErrJWTAccessTokenNotFound) Error() string {
	return "error code: 005 - message: invalid credentials"
}

type ErrJWTAccessTokenValidationError struct{}

func (e *ErrJWTAccessTokenValidationError) Error() string {
	return "error code: 006 - message: invalid credentials"
}

type ErrJWTAccessTokenCreateError struct{}

func (e *ErrJWTAccessTokenCreateError) Error() string {
	return "error code: 007 - message: fatal error"
}

type ErrJWTRefreshTokenCreateError struct{}

func (e *ErrJWTRefreshTokenCreateError) Error() string {
	return "error code: 008 - message: fatal error"
}

type ErrJWTRefreshTokenNotFound struct{}

func (e *ErrJWTRefreshTokenNotFound) Error() string {
	return "error code: 009 - message: invalid credentials"
}

type ErrJWTRefreshTokenValidationError struct{}

func (e *ErrJWTRefreshTokenValidationError) Error() string {
	return "error code: 010 - message: invalid credentials"
}

type ErrJWTRefreshTokenExpired struct{}

func (e *ErrJWTRefreshTokenExpired) Error() string {
	return "error code: 011 - message: invalid credentials"
}
