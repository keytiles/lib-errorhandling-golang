package kt_error_test

import (
	"fmt"
	"testing"

	"github.com/keytiles/lib-errorhandling-golang/v2/pkg/kt_errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

var allFaultKinds = []kt_errors.FaultKind{
	kt_errors.RuntimeFault,
	kt_errors.AuthenticationFault,
	kt_errors.AuthorizationFault,
	kt_errors.ConstraintViolationFault,
	kt_errors.IllegalStateFault,
	kt_errors.NotImplementedFault,
	kt_errors.ResourceNotFoundFault,
	kt_errors.ValidationFault,
}

// Verifies HTTP status mapping: nil, non-public → 500, public kind defaults, and error-code overrides (+ Fault wrapper).
func TestHttpStatusCodeFromFault(t *testing.T) {

	var fault kt_errors.Fault

	// ==================
	// Scenario 1
	// ==================
	// Nil fault - should return OK

	// ---- WHEN
	statusCode := kt_errors.GetHttpStatusCodeForFault(nil)
	// ---- THEN
	assert.Equal(t, 200, statusCode)

	// ==================
	// Scenario 2
	// ==================
	// All and any non-public Fault - should return 500

	for _, faultKind := range allFaultKinds {
		// ---- WHEN
		fault = kt_errors.NewFaultBuilder(faultKind).Build()
		statusCode = kt_errors.GetHttpStatusCodeForFault(fault)
		// ---- THEN
		assert.Equal(t, 500, statusCode)
	}

	// ==================
	// Scenario 3
	// ==================
	// Verify default mapping (only based on Kind - no error codes) of public FaultKinds

	// ---- GIVEN
	pubFaultKindsDefaultHttpStatuses := map[kt_errors.FaultKind]int{
		kt_errors.RuntimeFault:             500,
		kt_errors.AuthenticationFault:      401,
		kt_errors.AuthorizationFault:       403,
		kt_errors.ConstraintViolationFault: 412,
		kt_errors.IllegalStateFault:        500,
		kt_errors.NotImplementedFault:      501,
		kt_errors.ResourceNotFoundFault:    404,
		kt_errors.ValidationFault:          400,
	}

	for faultKind, expectedStatus := range pubFaultKindsDefaultHttpStatuses {
		// ---- WHEN
		fault = kt_errors.NewPublicFaultBuilder(faultKind).Build()
		statusCode = kt_errors.GetHttpStatusCodeForFault(fault)
		// ---- THEN
		assert.Equal(t, expectedStatus, statusCode, fmt.Sprintf("Fault kind '%s' did not return expected http status code", faultKind))
	}

	// ==================
	// Scenario 4
	// ==================
	// Error-code overrides for public Faults (ConstraintViolation / IllegalState)

	// ---- GIVEN
	httpOverrideCases := []struct {
		kind     kt_errors.FaultKind
		errCode  string
		expected int
	}{
		{kt_errors.ConstraintViolationFault, kt_errors.CONSTRAINTVIOLATION_ERRCODE_ID_ALREADY_TAKEN, 409},
		{kt_errors.ConstraintViolationFault, kt_errors.CONSTRAINTVIOLATION_ERRCODE_ALREADY_EXIST, 409},
		{kt_errors.ConstraintViolationFault, kt_errors.CONSTRAINTVIOLATION_ERRCODE_DOES_NOT_EXIST, 404},
		{kt_errors.IllegalStateFault, kt_errors.ILLEGALSTATE_ERRCODE_DEPENDENCY_UNAVAILABLE, 503},
		{kt_errors.IllegalStateFault, kt_errors.ILLEGALSTATE_ERRCODE_EXHAUSTED, 503},
		{kt_errors.IllegalStateFault, kt_errors.ILLEGALSTATE_ERRCODE_TIMED_OUT, 503},
		{kt_errors.IllegalStateFault, kt_errors.ILLEGALSTATE_ERRCODE_EXPECTATION_FAILED, 412},
	}

	for _, tc := range httpOverrideCases {
		// ---- GIVEN
		fault = kt_errors.NewPublicFaultBuilder(tc.kind).
			WithErrorCodes(tc.errCode).
			Build()
		// ---- WHEN
		statusCode = kt_errors.GetHttpStatusCodeForFault(fault)
		wrapperStatus := fault.GetHttpStatusCode()
		// ---- THEN
		assert.Equal(t, tc.expected, statusCode, "kind=%s errCode=%s", tc.kind, tc.errCode)
		assert.Equal(t, statusCode, wrapperStatus, "Fault.GetHttpStatusCode must match GetHttpStatusCodeForFault")
	}

	// ==================
	// Scenario 5
	// ==================
	// Deprecated misspelled alias still maps the same (same string value as EXPECTATION_FAILED).

	// ---- GIVEN
	assert.Equal(t, kt_errors.ILLEGALSTATE_ERRCODE_EXPECTATION_FAILED, kt_errors.ILLEGALSTATE_ERRCODE_EXCPECTATION_FAILED)
	fault = kt_errors.NewPublicFaultBuilder(kt_errors.IllegalStateFault).
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_EXCPECTATION_FAILED).
		Build()
	// ---- WHEN
	statusCode = kt_errors.GetHttpStatusCodeForFault(fault)
	// ---- THEN
	assert.Equal(t, 412, statusCode)
}

// Verifies gRPC status mapping: nil, non-public → Internal, public kind defaults, and error-code overrides (+ Fault wrapper).
func TestGrpcStatusCodeFromFault(t *testing.T) {

	var fault kt_errors.Fault

	// ==================
	// Scenario 1
	// ==================
	// Nil fault - should return OK

	// ---- WHEN
	statusCode := kt_errors.GetGrpcStatusCodeForFault(nil)
	// ---- THEN
	assert.Equal(t, codes.OK, statusCode)

	// ==================
	// Scenario 2
	// ==================
	// All and any non-public Fault - should return Internal error

	// ---- GIVEN

	for _, faultKind := range allFaultKinds {
		// ---- WHEN
		fault = kt_errors.NewFaultBuilder(faultKind).Build()
		statusCode = kt_errors.GetGrpcStatusCodeForFault(fault)
		// ---- THEN
		assert.Equal(t, codes.Internal, statusCode)
	}

	// ==================
	// Scenario 3
	// ==================
	// Verify default mapping (only based on Kind - no error codes) of public FaultKinds

	// ---- GIVEN
	pubFaultKindsDefaultGrpcStatuses := map[kt_errors.FaultKind]codes.Code{
		kt_errors.RuntimeFault:             codes.Internal,
		kt_errors.AuthenticationFault:      codes.Unauthenticated,
		kt_errors.AuthorizationFault:       codes.PermissionDenied,
		kt_errors.ConstraintViolationFault: codes.FailedPrecondition,
		kt_errors.IllegalStateFault:        codes.Internal,
		kt_errors.NotImplementedFault:      codes.Unimplemented,
		kt_errors.ResourceNotFoundFault:    codes.NotFound,
		kt_errors.ValidationFault:          codes.InvalidArgument,
	}

	for faultKind, expectedStatus := range pubFaultKindsDefaultGrpcStatuses {
		// ---- WHEN
		fault = kt_errors.NewPublicFaultBuilder(faultKind).Build()
		statusCode = kt_errors.GetGrpcStatusCodeForFault(fault)
		// ---- THEN
		assert.Equal(t, expectedStatus, statusCode, fmt.Sprintf("Fault kind '%s' did not return expected grpc status code", faultKind))
	}

	// ==================
	// Scenario 4
	// ==================
	// Error-code overrides for public Faults (ConstraintViolation / IllegalState)

	// ---- GIVEN
	grpcOverrideCases := []struct {
		kind     kt_errors.FaultKind
		errCode  string
		expected codes.Code
	}{
		{kt_errors.ConstraintViolationFault, kt_errors.CONSTRAINTVIOLATION_ERRCODE_ID_ALREADY_TAKEN, codes.AlreadyExists},
		{kt_errors.ConstraintViolationFault, kt_errors.CONSTRAINTVIOLATION_ERRCODE_ALREADY_EXIST, codes.AlreadyExists},
		{kt_errors.ConstraintViolationFault, kt_errors.CONSTRAINTVIOLATION_ERRCODE_DOES_NOT_EXIST, codes.NotFound},
		{kt_errors.IllegalStateFault, kt_errors.ILLEGALSTATE_ERRCODE_DEPENDENCY_UNAVAILABLE, codes.Unavailable},
		{kt_errors.IllegalStateFault, kt_errors.ILLEGALSTATE_ERRCODE_TIMED_OUT, codes.Unavailable},
		{kt_errors.IllegalStateFault, kt_errors.ILLEGALSTATE_ERRCODE_EXHAUSTED, codes.ResourceExhausted},
		{kt_errors.IllegalStateFault, kt_errors.ILLEGALSTATE_ERRCODE_EXPECTATION_FAILED, codes.FailedPrecondition},
	}

	for _, tc := range grpcOverrideCases {
		// ---- GIVEN
		fault = kt_errors.NewPublicFaultBuilder(tc.kind).
			WithErrorCodes(tc.errCode).
			Build()
		// ---- WHEN
		statusCode = kt_errors.GetGrpcStatusCodeForFault(fault)
		wrapperStatus := fault.GetGrpcStatusCode()
		// ---- THEN
		assert.Equal(t, tc.expected, statusCode, "kind=%s errCode=%s", tc.kind, tc.errCode)
		assert.Equal(t, statusCode, wrapperStatus, "Fault.GetGrpcStatusCode must match GetGrpcStatusCodeForFault")
	}
}

func TestGetFaultAsNaturalJSON_NilFaultIsSafe(t *testing.T) {
	// Nil Fault must serialize to the empty/NaN natural JSON form without error.

	// ---- GIVEN
	var fault kt_errors.Fault = nil

	// ---- WHEN
	json, err := kt_errors.GetFaultAsNaturalJSON(fault, "")
	// ---- THEN
	assert.NoError(t, err)
	assert.Equal(t, `{"kind":"NaN","message":"","isRetryable":false,"errorCodes":[],"labels":{}}`, string(json))
}

func TestGetFaultAsFullJSON_NilFaultIsSafe(t *testing.T) {
	// Nil Fault must serialize to the empty/NaN full JSON form; ResolveMessages must not panic.

	// ---- GIVEN
	var fault kt_errors.Fault = nil

	// ---- WHEN
	json, err := kt_errors.GetFaultAsFullJSON(fault)
	// ---- THEN
	assert.NoError(t, err)
	assert.Equal(t, `{"kind":"NaN","message":"","messagesByAudience":{},"isRetryable":false,"errorCodes":[],"labels":{}}`, string(json))

	// ---- WHEN
	json, err = kt_errors.GetFaultAsFullJSON(fault, kt_errors.ResolveMessages)
	// ---- THEN
	assert.NoError(t, err)
	assert.Equal(t, `{"kind":"NaN","message":"","messagesByAudience":{},"isRetryable":false,"errorCodes":[],"labels":{}}`, string(json))
}

func TestNewPublicFaultFromAnyError_PlainErrorAndNilOptionsAreSafe(t *testing.T) {
	// Plain non-Fault error + inheritErrorCodes must not panic; nil ConversionOption is skipped; nil original → nil.

	// ---- GIVEN
	plainErr := fmt.Errorf("plain boom")

	// ---- WHEN
	converted := kt_errors.NewPublicFaultFromAnyError(
		plainErr,
		"trId",
		nil,
		kt_errors.OptionWhitelistedFaultKinds(true, kt_errors.ValidationFault),
		nil, // nil ConversionOption must be skipped safely
	)
	// ---- THEN
	assert.True(t, converted.IsPublic())
	assert.Equal(t, kt_errors.RuntimeFault, converted.GetKind())
	assert.True(t, converted.HasErrorCode(kt_errors.ERRCODE_INTERNAL_ERROR))
	assert.Equal(t, plainErr, converted.GetCause())

	// ---- WHEN / THEN
	assert.Nil(t, kt_errors.NewPublicFaultFromAnyError(nil, "", nil))
}

func TestTypedNilFault_ErrorAndStringAreSafe(t *testing.T) {
	// Typed-nil Fault (interface holding nil *defaultFault) must not panic on Error()/String().

	// ---- GIVEN
	fault := kt_errors.VisibleForTesting_NilFault()

	// ---- WHEN / THEN
	assert.Equal(t, "", fault.Error())
	assert.Equal(t, "Fault{nil}", fault.String())
}

// IsFault distinguishes nil, plain errors, and Fault instances.
func TestIsFault(t *testing.T) {

	// ==================
	// Scenario 1
	// ==================
	// Nil error is not a Fault.

	// ---- WHEN
	ok, fault := kt_errors.IsFault(nil)
	// ---- THEN
	assert.False(t, ok)
	assert.Nil(t, fault)

	// ==================
	// Scenario 2
	// ==================
	// Plain error is not a Fault.

	// ---- GIVEN
	plain := fmt.Errorf("plain")
	// ---- WHEN
	ok, fault = kt_errors.IsFault(plain)
	// ---- THEN
	assert.False(t, ok)
	assert.Nil(t, fault)

	// ==================
	// Scenario 3
	// ==================
	// Built Fault is detected and returned.

	// ---- GIVEN
	original := kt_errors.NewPublicFaultBuilder(kt_errors.RuntimeFault).
		WithMessageTemplate("x").
		Build()
	// ---- WHEN
	ok, fault = kt_errors.IsFault(original)
	// ---- THEN
	assert.True(t, ok)
	assert.Equal(t, original, fault)
}
