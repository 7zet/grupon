package httpapi

import (
	"errors"
	"net/http"

	"github.com/grupon/grupon/services/api/internal/http/gen"
)

// bindingErrorHandler reports malformed path or query input as an RFC 9457 problem.
func bindingErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var required *gen.RequiredParamError
	var invalid *gen.InvalidParamFormatError
	switch {
	case errors.As(err, &required):
		writeProblem(w, r, http.StatusBadRequest, "required_parameter_missing", nil,
			map[string][]string{required.ParamName: {"required"}})
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusBadRequest, "invalid_parameter", nil,
			map[string][]string{invalid.ParamName: {"invalid_format"}})
	default:
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", nil, nil)
	}
}
