package models

import (
	"net/http"
)

var RetryableHTTPCodes = map[int]struct{}{
	http.StatusBadGateway:          {},
	http.StatusServiceUnavailable:  {},
	http.StatusInternalServerError: {},
}
