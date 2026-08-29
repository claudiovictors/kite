package kite

import (
	"fmt"
	"net/http"
)

// HTTPError representa um erro HTTP estruturado dentro da framework Kite.
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("kite: status %d - %s", e.Code, e.Message)
}

// NewError cria um novo HTTPError preenchendo a mensagem com a descrição padrão HTTP do status code.
func NewError(code int, message ...string) *HTTPError {
	msg := http.StatusText(code)
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return &HTTPError{
		Code:    code,
		Message: msg,
	}
}

// Erros pré-definidos reutilizáveis
var (
	ErrBadRequest                    = NewError(StatusBadRequest)                    // 400
	ErrUnauthorized                  = NewError(StatusUnauthorized)                  // 401
	ErrPaymentRequired               = NewError(StatusPaymentRequired)               // 402
	ErrForbidden                     = NewError(StatusForbidden)                     // 403
	ErrNotFound                      = NewError(StatusNotFound)                      // 404
	ErrMethodNotAllowed              = NewError(StatusMethodNotAllowed)              // 405
	ErrNotAcceptable                 = NewError(StatusNotAcceptable)                 // 406
	ErrProxyAuthRequired             = NewError(StatusProxyAuthRequired)             // 407
	ErrRequestTimeout                = NewError(StatusRequestTimeout)                // 408
	ErrConflict                      = NewError(StatusConflict)                      // 409
	ErrGone                          = NewError(StatusGone)                          // 410
	ErrLengthRequired                = NewError(StatusLengthRequired)                // 411
	ErrPreconditionFailed            = NewError(StatusPreconditionFailed)            // 412
	ErrRequestEntityTooLarge         = NewError(StatusRequestEntityTooLarge)         // 413
	ErrRequestURITooLong             = NewError(StatusRequestURITooLong)             // 414
	ErrUnsupportedMediaType          = NewError(StatusUnsupportedMediaType)          // 415
	ErrRequestedRangeNotSatisfiable  = NewError(StatusRequestedRangeNotSatisfiable)  // 416
	ErrExpectationFailed             = NewError(StatusExpectationFailed)             // 417
	ErrTeapot                        = NewError(StatusTeapot)                        // 418
	ErrMisdirectedRequest            = NewError(StatusMisdirectedRequest)            // 421
	ErrUnprocessableEntity           = NewError(StatusUnprocessableEntity)           // 422
	ErrLocked                        = NewError(StatusLocked)                        // 423
	ErrFailedDependency              = NewError(StatusFailedDependency)              // 424
	ErrTooEarly                      = NewError(StatusTooEarly)                      // 425
	ErrUpgradeRequired               = NewError(StatusUpgradeRequired)               // 426
	ErrPreconditionRequired          = NewError(StatusPreconditionRequired)          // 428
	ErrTooManyRequests               = NewError(StatusTooManyRequests)               // 429
	ErrRequestHeaderFieldsTooLarge   = NewError(StatusRequestHeaderFieldsTooLarge)   // 431
	ErrUnavailableForLegalReasons    = NewError(StatusUnavailableForLegalReasons)    // 451
	ErrInternalServerError           = NewError(StatusInternalServerError)           // 500
	ErrNotImplemented                = NewError(StatusNotImplemented)                // 501
	ErrBadGateway                    = NewError(StatusBadGateway)                    // 502
	ErrServiceUnavailable            = NewError(StatusServiceUnavailable)            // 503
	ErrGatewayTimeout                = NewError(StatusGatewayTimeout)                // 504
	ErrHTTPVersionNotSupported       = NewError(StatusHTTPVersionNotSupported)       // 505
	ErrVariantAlsoNegotiates         = NewError(StatusVariantAlsoNegotiates)         // 506
	ErrInsufficientStorage           = NewError(StatusInsufficientStorage)           // 507
	ErrLoopDetected                  = NewError(StatusLoopDetected)                  // 508
	ErrNotExtended                   = NewError(StatusNotExtended)                   // 510
	ErrNetworkAuthenticationRequired = NewError(StatusNetworkAuthenticationRequired) // 511
)