package kt_error_test

import (
	"fmt"
	"testing"

	"github.com/keytiles/lib-errorhandling-golang/pkg/kt_errors"
	"github.com/keytiles/lib-utils-golang/pkg/kt_utils"
	"github.com/stretchr/testify/assert"
)

func TestNonPublicBuilderAndFault(t *testing.T) {

	// ---- GIVEN
	cause := fmt.Errorf("cause error")
	builder := kt_errors.NewFaultBuilder(kt_errors.IllegalStateFault).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithMessageTemplateForAudience(kt_errors.MSGAUDIENCE_USER, "user message with var={var1}").
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR, "remove_this").
		WithoutErrorCodes("remove_this").
		WithLabel("var1", "value1").
		WithLabels(map[string]any{"var2": "value2", "var3": "value3"}).
		WithoutLabels("var2", "var3").
		WithSource("mymodule", "myfunction").
		WithCause(cause)

	// ---- WHEN
	fault := builder.Build()
	// ---- THEN
	assert.Error(t, fault)
	assert.Equal(t, kt_errors.IllegalStateFault, fault.GetKind())
	assert.False(t, fault.IsPublic())
	assert.False(t, fault.IsRetryable())
	assert.Equal(t, cause, fault.GetCause())
	assert.Equal(t, "message with var=value1 and unknown {unknown_var}", fault.GetMessage())
	assert.Equal(t, "user message with var=value1", fault.GetMessageForAudience(kt_errors.MSGAUDIENCE_USER))
	assert.Equal(t, "", fault.GetMessageForAudience("not-set"))
	assert.Equal(t, 1, len(fault.GetErrorCodes()))
	assert.True(t, fault.HasErrorCode(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR))
	assert.Equal(t, map[string]any{"var1": "value1"}, fault.GetLabels())

	// ---- GIVEN
	// let's add a caller to the stack!
	fault.AddCallerToCallStack("mycallermodule", "mycallerfunction")
	// ---- WHEN
	callStack := fault.GetCallStack()
	// ---- THEN
	assert.Equal(t, []string{"mycallermodule.mycallerfunction", "mymodule.myfunction"}, callStack)

	// let's test to string and standard error

	// ---- WHEN
	error_query_result := fault.Error()
	// ---- THEN
	assert.Equal(
		t,
		"illegal_state: message with var=value1 and unknown {unknown_var} (retryable: false, errorCodes: ['config_error'])",
		error_query_result,
	)

	// ---- WHEN
	tostring_result := fault.String()
	// ---- THEN
	assert.Equal(
		t,
		"Fault{type: 'illegal_state', msgTemplate: 'message with var={var1} and unknown {unknown_var}', retryable: false, public: false, codes: ['config_error'], callStack: ['mycallermodule.mycallerfunction','mymodule.myfunction'], cause: 'cause error', audienceMsgs: map[string]string{\"user\":\"user message with var={var1}\"}, labels: map[string]interface{}{\"var1\":\"value1\"}}",
		tostring_result,
	)

	// ---- WHEN
	// printing into a string with %s
	str_s := fmt.Sprintf("error printing: %s", fault)
	// ---- THEN
	// since this is an error (interface) the .Error() function is used in above case by Go
	assert.Equal(t, "error printing: "+fault.Error(), str_s)

	// ---- WHEN
	// printing into a string with %v - should be the same actually as printing with %s
	str_v := fmt.Sprintf("error printing: %v", fault)
	// ---- THEN
	// since this is an error (interface) the .Error() function is used in above case by Go
	assert.Equal(t, str_s, str_v)

	// ---- WHEN
	// printing into a string with %+v - should be the same actually as printing with %s
	str_pv := fmt.Sprintf("error printing: %+v", fault)
	// ---- THEN
	// since this is an error (interface) the .Error() function is used in above case by Go
	assert.Equal(t, str_s, str_pv)

	// ---- WHEN
	// printing into a string with VarPrinter - this should use the toString() method correctly
	str_varp := fmt.Sprintf("error printing: %s", kt_utils.VarPrinter{TheVar: fault})
	// ---- THEN
	// since this is an error (interface) the .Error() function is used in above case by Go
	assert.Equal(t, "error printing: "+fault.String(), str_varp)
}

func TestPublicBuilderAndFault(t *testing.T) {

	// ---- GIVEN
	cause := fmt.Errorf("cause error")
	builder := kt_errors.NewPublicFaultBuilder(kt_errors.IllegalStateFault).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithIsRetryable(true).
		WithErrorCodes("internal_error", "remove_this").
		WithoutErrorCodes("remove_this").
		WithLabel("var1", "value1").
		WithLabels(map[string]any{"var2": "value2", "var3": "value3"}).
		WithoutLabels("var2", "var3").
		WithCause(cause).
		WithoutCause()

	// ---- WHEN
	fault := builder.Build()
	// ---- THEN
	assert.Error(t, fault)
	assert.Equal(t, kt_errors.IllegalStateFault, fault.GetKind())
	assert.True(t, fault.IsPublic())
	assert.True(t, fault.IsRetryable())
	assert.Nil(t, fault.GetCause())
	assert.Equal(t, "message with var=value1 and unknown {unknown_var}", fault.GetMessage())
	assert.Equal(t, 1, len(fault.GetErrorCodes()))
	assert.True(t, fault.HasErrorCode("internal_error"))
	assert.Equal(t, map[string]any{"var1": "value1"}, fault.GetLabels())
	assert.Equal(t, "", fault.GetSource())
	assert.Equal(t, 0, len(fault.GetCallStack()))

	// let's test to string and standard error

	// ---- WHEN
	error_query_result := fault.Error()
	// ---- THEN
	assert.Equal(
		t,
		"illegal_state: message with var=value1 and unknown {unknown_var} (retryable: true, errorCodes: ['internal_error'], labels: map[string]interface{}{\"var1\":\"value1\"})",
		error_query_result,
	)

	// ---- WHEN
	tostring_result := fault.String()
	// ---- THEN
	assert.Equal(
		t,
		"Fault{type: 'illegal_state', msgTemplate: 'message with var={var1} and unknown {unknown_var}', retryable: true, public: true, codes: ['internal_error'], callStack: [], cause: nil, audienceMsgs: map[string]string{}, labels: map[string]interface{}{\"var1\":\"value1\"}}",
		tostring_result,
	)

	// ---- WHEN
	str := fmt.Sprintf("error printing: %s", fault)
	// ---- THEN
	// since this is an error (interface) the .Error() function is used in above case by Go
	assert.Equal(t, "error printing: "+fault.Error(), str)
}

func TestPublicFaultCreation_fromPublicFault(t *testing.T) {

	// ---- GIVEN
	cause := fmt.Errorf("cause error")
	originalErr := kt_errors.NewPublicFaultBuilder(kt_errors.IllegalStateFault).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithIsRetryable(true).
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR, "internal_error").
		WithLabel("var1", "value1").
		WithCause(cause).
		Build()

	// ---- WHEN
	converted := kt_errors.NewPublicFaultFromAnyError(originalErr, "trId", nil, kt_errors.NO_LOG_LABELS)

	// ---- THEN
	// since original error was already public, it is returned as it is
	assert.Equal(t, originalErr, converted)
}

func TestPublicFaultCreation_fromNonPublicFaultError(t *testing.T) {

	// ===============================
	// Scenario 1
	// ===============================
	// The Fault does not contain any audience message template

	// ---- GIVEN
	cause := fmt.Errorf("cause error")
	originalFault := kt_errors.NewFaultBuilder(kt_errors.IllegalStateFault).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithIsRetryable(true).
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR, "some_other_error").
		WithLabel("var1", "value1").
		WithCause(cause).
		Build()

	// ---- WHEN
	converted := kt_errors.NewPublicFaultFromAnyError(originalFault, "trId", nil, kt_errors.NO_LOG_LABELS)

	// ---- THEN
	// what we got is public
	assert.True(t, converted.IsPublic())
	// the original error is added as a cause - as it is
	assert.Equal(t, originalFault, converted.GetCause())
	// the type is always RuntimeError
	assert.Equal(t, kt_errors.RuntimeFault, converted.GetKind())
	// error codes removed - but marked as internal error
	assert.Equal(t, 1, len(converted.GetErrorCodes()))
	assert.True(t, converted.HasErrorCode(kt_errors.ERRCODE_INTERNAL_ERROR))
	// but "isRetryable inherited"
	assert.True(t, converted.IsRetryable())
	// the message is strict - containing the transaction id as we had ExecutionContext
	assert.Equal(t, "Error occured during processing, details are logged with transactionId '{transactionId}'", converted.GetMessageTemplate())
	// and transactionId is added as label
	assert.Equal(t, map[string]any{"transactionId": "trId"}, converted.GetLabels())
	// there are no audience messages in the exception
	assert.Empty(t, converted.GetMessageTemplatesByAudience())

	// ===============================
	// Scenario 2
	// ===============================
	// The Fault contains some audience message templates - but not "user" facing

	// ---- GIVEN
	cause = fmt.Errorf("cause error")
	originalFault = kt_errors.NewFaultBuilder(kt_errors.IllegalStateFault).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithMessageTemplateForAudience("audience1", "message with {var2}").
		WithMessageTemplateForAudience("audience2", "message with {var2} and {var3}").
		WithIsRetryable(true).
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR, "some_other_error").
		WithLabel("var1", "value1").
		WithLabel("var2", "value2").
		WithLabel("var3", "value3").
		WithCause(cause).
		Build()

	// ---- WHEN
	converted = kt_errors.NewPublicFaultFromAnyError(originalFault, "trId", nil, kt_errors.NO_LOG_LABELS)

	// ---- THEN
	// what we got is public
	assert.True(t, converted.IsPublic())
	// the original error is added as a cause - as it is
	assert.Equal(t, originalFault, converted.GetCause())
	// the type is always RuntimeError
	assert.Equal(t, kt_errors.RuntimeFault, converted.GetKind())
	// error codes removed - but marked as internal error
	assert.Equal(t, 1, len(converted.GetErrorCodes()))
	assert.True(t, converted.HasErrorCode(kt_errors.ERRCODE_INTERNAL_ERROR))
	// but "isRetryable inherited"
	assert.True(t, converted.IsRetryable())
	// the message is strict - containing the transaction id as we had ExecutionContext
	assert.Equal(t, "Error occured during processing, details are logged with transactionId '{transactionId}'", converted.GetMessageTemplate())
	// The audience messages should be inherited
	assert.Equal(t, originalFault.GetMessageTemplatesByAudience(), converted.GetMessageTemplatesByAudience())
	// and transactionId is added as label plus we kept all labels needed to resolve audience messages
	assert.Equal(t, map[string]any{"transactionId": "trId", "var2": "value2", "var3": "value3"}, converted.GetLabels())

	// ===============================
	// Scenario 3
	// ===============================
	// The Fault contains some audience AND also contains "user" facing
	// That

	// ---- GIVEN
	cause = fmt.Errorf("cause error")
	originalFault = kt_errors.NewFaultBuilder(kt_errors.IllegalStateFault).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithMessageTemplateForAudience(kt_errors.MSGAUDIENCE_USER, "user facing message with {var1}").
		WithMessageTemplateForAudience("audience1", "message with {var3}").
		WithIsRetryable(true).
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR, "some_other_error").
		WithLabel("var1", "value1").
		WithLabel("var2", "value2").
		WithLabel("var3", "value3").
		WithCause(cause).
		Build()

	// ---- WHEN
	converted = kt_errors.NewPublicFaultFromAnyError(originalFault, "trId", nil, kt_errors.NO_LOG_LABELS)

	// ---- THEN
	// what we got is public
	assert.True(t, converted.IsPublic())
	// the original error is added as a cause - as it is
	assert.Equal(t, originalFault, converted.GetCause())
	// the type is always RuntimeError
	assert.Equal(t, kt_errors.RuntimeFault, converted.GetKind())
	// error codes removed - but marked as internal error
	assert.Equal(t, 1, len(converted.GetErrorCodes()))
	assert.True(t, converted.HasErrorCode(kt_errors.ERRCODE_INTERNAL_ERROR))
	// but "isRetryable inherited"
	assert.True(t, converted.IsRetryable())
	// the message should be inherited from MSGAUDIENCE_USER audience template
	assert.Equal(t, "user facing message with {var1}", converted.GetMessageTemplate())
	// Only one audience message remains
	assert.Equal(t, map[string]string{"audience1": "message with {var3}"}, converted.GetMessageTemplatesByAudience())
	// and we kept all labels needed to resolve audience messages
	assert.Equal(t, map[string]any{"var1": "value1", "var3": "value3"}, converted.GetLabels())

}

func TestToString_causeIsAnotherFault(t *testing.T) {

	// ---- GIVEN

	// we create a Fault - non-public
	originalFault := kt_errors.NewFaultBuilder(kt_errors.IllegalStateFault).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithIsRetryable(true).
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR, "some_other_error").
		WithLabel("var1", "value1").
		WithSource("mymodule", "myfunction").
		Build()
	// and convert it into a public one - this will set the 'cause' to the original
	converted := kt_errors.NewPublicFaultFromAnyError(originalFault, "trId", nil, kt_errors.NO_LOG_LABELS)

	// ---- WHEN
	tostring_result := converted.String()
	// ---- THEN
	// we should see the original error appearing as cause in the output wit full toString()
	assert.Contains(t, tostring_result, fmt.Sprintf("cause: {%s}", originalFault.String()))
}

func TestAddingMoreContextToFault(t *testing.T) {

	// ---- GIVEN

	// we create a Fault - non-public but this does not matter now much
	fault := kt_errors.NewFaultBuilder(kt_errors.IllegalStateFault).
		WithIsRetryable(true).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithMessageTemplateForAudience("operator", "message for operators").
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR, "some_other_error").
		WithLabel("var1", "value1").
		WithSource("mymodule", "myfunction").
		Build()

	// ---- WHEN
	fault.AddContextToMessage("added msg context with {var2} - ")
	fault.AddContextToAudienceMessage("operator", "added audience msg context {var3} - ")
	fault.AddContextToAudienceMessage("new_audience1", "added total new audience msg context {var3} - ")
	fault.AddContextToAudienceMessage("new_audience2", "added total new audience msg context but with colon: ")
	fault.AddLabel("var2", "var2value")
	fault.AddErrorCodes("amended_err_code")

	// ---- THEN

	// message of the error concatenated correctly
	assert.Equal(t, "added msg context with {var2} - message with var={var1} and unknown {unknown_var}", fault.GetMessageTemplate())
	// and also, the new "var2" label is really there - so message resolves as it should
	assert.Equal(t, "added msg context with var2value - message with var=value1 and unknown {unknown_var}", fault.GetMessage())

	// audience messages now have a new entry
	assert.Equal(t, 3, len(fault.GetMessageTemplatesByAudience()))
	// the existing one prepended
	assert.Equal(t, "added audience msg context {var3} - message for operators", fault.GetMessageTemplateForAudience("operator"))
	// but the new one stays as is - trimmed on the right
	assert.Equal(t, "added total new audience msg context {var3}", fault.GetMessageTemplateForAudience("new_audience1"))
	assert.Equal(t, "added total new audience msg context but with colon", fault.GetMessageTemplateForAudience("new_audience2"))

	// error codes are extended too
	assert.Equal(t, 3, len(fault.GetErrorCodes()))
	assert.True(t, fault.HasErrorCode("amended_err_code"))
}
