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
}

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
	pubFaultKindsDefaultHttpStatuses := map[kt_errors.FaultKind]codes.Code{
		kt_errors.RuntimeFault:             codes.Internal,
		kt_errors.AuthenticationFault:      codes.Unauthenticated,
		kt_errors.AuthorizationFault:       codes.PermissionDenied,
		kt_errors.ConstraintViolationFault: codes.FailedPrecondition,
		kt_errors.IllegalStateFault:        codes.Internal,
		kt_errors.NotImplementedFault:      codes.Unimplemented,
		kt_errors.ResourceNotFoundFault:    codes.NotFound,
		kt_errors.ValidationFault:          codes.InvalidArgument,
	}

	for faultKind, expectedStatus := range pubFaultKindsDefaultHttpStatuses {
		// ---- WHEN
		fault = kt_errors.NewPublicFaultBuilder(faultKind).Build()
		statusCode = kt_errors.GetGrpcStatusCodeForFault(fault)
		// ---- THEN
		assert.Equal(t, expectedStatus, statusCode, fmt.Sprintf("Fault kind '%s' did not return expected grpc status code", faultKind))
	}

}

func TestGetFaultAsNaturalJSON_NilFaultIsSafe(t *testing.T) {
	var fault kt_errors.Fault = nil

	json, err := kt_errors.GetFaultAsNaturalJSON(fault, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"kind":"NaN","message":"","isRetryable":false,"errorCodes":[],"labels":{}}`, string(json))
}

func TestGetFaultAsFullJSON_NilFaultIsSafe(t *testing.T) {
	var fault kt_errors.Fault = nil

	json, err := kt_errors.GetFaultAsFullJSON(fault)
	assert.NoError(t, err)
	assert.Equal(t, `{"kind":"NaN","message":"","messagesByAudience":{},"isRetryable":false,"errorCodes":[],"labels":{}}`, string(json))

	// ResolveMessages on nil must not panic (previously dereferenced nil receiver in resolve path)
	json, err = kt_errors.GetFaultAsFullJSON(fault, kt_errors.ResolveMessages)
	assert.NoError(t, err)
	assert.Equal(t, `{"kind":"NaN","message":"","messagesByAudience":{},"isRetryable":false,"errorCodes":[],"labels":{}}`, string(json))
}

func TestNewPublicFaultFromAnyError_PlainErrorAndNilOptionsAreSafe(t *testing.T) {
	// ---- GIVEN
	plainErr := fmt.Errorf("plain boom")

	// ---- WHEN / THEN
	// inheritErrorCodes=true on a non-Fault must not panic
	converted := kt_errors.NewPublicFaultFromAnyError(
		plainErr,
		"trId",
		nil,
		kt_errors.OptionWhitelistedFaultKinds(true, kt_errors.ValidationFault),
		nil, // nil ConversionOption must be skipped safely
	)
	assert.True(t, converted.IsPublic())
	assert.Equal(t, kt_errors.RuntimeFault, converted.GetKind())
	assert.True(t, converted.HasErrorCode(kt_errors.ERRCODE_INTERNAL_ERROR))
	assert.Equal(t, plainErr, converted.GetCause())

	// nil original → nil
	assert.Nil(t, kt_errors.NewPublicFaultFromAnyError(nil, "", nil))
}

func TestTypedNilFault_ErrorAndStringAreSafe(t *testing.T) {
	// Typed-nil: Fault interface holding a nil *defaultFault (not a nil interface).
	// testify NotNil treats typed-nil as nil, so we only assert no panic + stable strings.
	fault := kt_errors.VisibleForTesting_NilFault()
	assert.Equal(t, "", fault.Error())
	assert.Equal(t, "Fault{nil}", fault.String())
}
