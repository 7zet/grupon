package httpapi

import (
	"errors"
	"net/http"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

// RequestPhoneCode sends a one-time sign-in code to the phone. The response never reveals whether
// the phone already has an account.
func (a *AuthAPI) RequestPhoneCode(w http.ResponseWriter, r *http.Request) {
	var body gen.PhoneCodeRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	requested, err := a.identity.RequestCode(r.Context(), a.meta(r), body.Phone)
	if err != nil {
		a.phoneFail(w, r, err)
		return
	}
	sent := gen.PhoneCodeSent{CooldownSeconds: requested.CooldownSeconds}
	if requested.DevCode != "" {
		code := requested.DevCode
		sent.DevCode = &code
	}
	writeJSON(w, http.StatusAccepted, sent)
}

// VerifyPhoneCode signs in with the code. The first successful code creates the account.
func (a *AuthAPI) VerifyPhoneCode(w http.ResponseWriter, r *http.Request) {
	var body gen.PhoneCodeVerify
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	result, err := a.identity.VerifyPhone(r.Context(), a.meta(r), body.Phone, body.Code)
	if err != nil {
		a.phoneFail(w, r, err)
		return
	}
	if result.Challenge != nil {
		writeJSON(w, http.StatusAccepted, gen.TwoFactorChallenge{ChallengeToken: result.Challenge.Token, EnrollmentRequired: result.Challenge.Enrollment})
		return
	}
	a.writeSession(w, http.StatusOK, *result.Session)
}

// phoneFail maps phone sign-in errors to stable problem codes; everything else goes to the shared mapping.
func (a *AuthAPI) phoneFail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, identity.ErrInvalidPhone):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_phone", nil,
			map[string][]string{"phone": {"invalid_phone"}})
	case errors.Is(err, identity.ErrOTPCooldown):
		writeProblem(w, r, http.StatusTooManyRequests, "otp_cooldown", nil, nil)
	case errors.Is(err, identity.ErrOTPInvalid):
		writeProblem(w, r, http.StatusUnauthorized, "otp_invalid", nil, nil)
	case errors.Is(err, identity.ErrOTPTooManyTries):
		writeProblem(w, r, http.StatusTooManyRequests, "otp_attempts_exceeded", nil, nil)
	case errors.Is(err, identity.ErrOTPUnavailable):
		writeProblem(w, r, http.StatusServiceUnavailable, "otp_unavailable", nil, nil)
	case errors.Is(err, identity.ErrSMSDailyCap):
		writeProblem(w, r, http.StatusTooManyRequests, "sms_daily_cap_reached", nil, nil)
	default:
		a.fail(w, r, err)
	}
}

// Delegation from the router.

func (s apiServer) RequestPhoneCode(w http.ResponseWriter, r *http.Request) {
	if s.auth != nil {
		s.auth.RequestPhoneCode(w, r)
		return
	}
	s.ServerInterface.RequestPhoneCode(w, r)
}

func (s apiServer) VerifyPhoneCode(w http.ResponseWriter, r *http.Request) {
	if s.auth != nil {
		s.auth.VerifyPhoneCode(w, r)
		return
	}
	s.ServerInterface.VerifyPhoneCode(w, r)
}
