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
