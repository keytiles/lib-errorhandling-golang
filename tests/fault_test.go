package kt_error_test

import (
	"fmt"
	"testing"

	"github.com/keytiles/lib-errorhandling-golang/pkg/kt_errors"
	"github.com/keytiles/lib-utils-golang/pkg/kt_utils"
	"github.com/stretchr/testify/assert"
)

func TestNonPublicBuilderAndFault(t *testing.T) {

	// ==================
	// Scenario 1
	// ==================
	// We go with super minimalistic info - will leave all arrays/maps Nil in the fault

	builder := kt_errors.NewFaultBuilder(kt_errors.IllegalStateFault).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}")

	// ---- WHEN
	fault := builder.Build()
	// ---- THEN
	assert.Error(t, fault)
	assert.Equal(t, kt_errors.IllegalStateFault, fault.GetKind())
	assert.False(t, fault.IsPublic())
	assert.False(t, fault.IsRetryable())
	assert.Nil(t, fault.GetCause())
	assert.Equal(t, "message with var={var1} and unknown {unknown_var}", fault.GetMessage())
	assert.Equal(t, "", fault.GetMessageForAudience("any"))
	assert.Equal(t, "", fault.GetMessageTemplateForAudience("any"))
	assert.Equal(t, 0, len(fault.GetErrorCodes()))
	assert.Equal(t, 0, len(fault.GetMessageTemplatesByAudience()))
	assert.Equal(t, 0, len(fault.GetLabels()))

	// ==================
	// Scenario 2
	// ==================
	// Now we go with full data rich stuff

	// ---- GIVEN
	cause := fmt.Errorf("cause error")
	builder = kt_errors.NewFaultBuilder(kt_errors.IllegalStateFault).
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
	fault = builder.Build()
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
	// ---- WHEN
	label, found := fault.GetLabel("not-exist")
	// ---- THEN
	assert.Nil(t, label)
	assert.False(t, found)
	// ---- WHEN
	label, found = fault.GetLabel("var1")
	// ---- THEN
	assert.Equal(t, "value1", label)
	assert.True(t, found)

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

	// now lets test that if we mutate the Fault that does not mutate the builder for sure and vice-versa

	// ---- WHEN
	fault.AddErrorCodes("new_err")
	fault.AddLabel("new_label", "buu")
	rebuiltFault := builder.Build()
	// ---- THEN
	assert.Equal(t, 1, len(rebuiltFault.GetErrorCodes()))
	assert.True(t, rebuiltFault.HasErrorCode(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR))
	assert.Equal(t, map[string]any{"var1": "value1"}, rebuiltFault.GetLabels())

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
		"Fault{type: 'illegal_state', msgTemplate: 'message with var={var1} and unknown {unknown_var}', retryable: true, public: true, codes: ['internal_error'], callStack: [], cause: nil, audienceMsgs: {}, labels: map[string]interface{}{\"var1\":\"value1\"}}",
		tostring_result,
	)

	// ---- WHEN
	str := fmt.Sprintf("error printing: %s", fault)
	// ---- THEN
	// since this is an error (interface) the .Error() function is used in above case by Go
	assert.Equal(t, "error printing: "+fault.Error(), str)

	// now lets test that if we mutate the Fault that does not mutate the builder for sure and vice-versa

	// ---- WHEN
	fault.AddErrorCodes("new_err")
	fault.AddLabel("new_label", "buu")
	rebuiltFault := builder.Build()
	// ---- THEN
	assert.Equal(t, 1, len(rebuiltFault.GetErrorCodes()))
	assert.True(t, rebuiltFault.HasErrorCode("internal_error"))
	assert.Equal(t, map[string]any{"var1": "value1"}, rebuiltFault.GetLabels())

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
	// and we kept all labels needed to resolve audience messages + the "transactionId"
	assert.Equal(t, map[string]any{"var1": "value1", "var3": "value3", "transactionId": "trId"}, converted.GetLabels())

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

func TestNonPublicFaultNaturalJSONSerialization(t *testing.T) {

	// ---- GIVEN

	// we create a Fault - non-public but this does not matter now much
	fault := kt_errors.NewFaultBuilder(kt_errors.IllegalStateFault).
		WithIsRetryable(true).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithMessageTemplateForAudience("operator", "message for operators").
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR).
		WithLabel("var1", "value1").
		WithSource("mymodule", "myfunction").
		Build()

	// ==================
	// Scenario 1
	// ==================
	// Non-public Fault should basically return empty things by default

	// ---- WHEN
	json, err := fault.ToNaturalJSON("")
	// ---- THEN
	assert.NoError(t, err)
	// no details should be exposed
	jsonStr := string(json)
	assert.Equal(
		t,
		`{"kind":"runtime","message":"","isRetryable":true,"errorCodes":[],"labels":{}}`,
		jsonStr,
	)

	// ==================
	// Scenario 2
	// ==================
	// But if we explicitly tell it is OK then it should return the details normally

	// ---- WHEN
	json, err = fault.ToNaturalJSON("", kt_errors.AllowNonPublicSerialization)
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr = string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message with var={var1} and unknown {unknown_var}","isRetryable":true,"errorCodes":["config_error"],"labels":{"var1":"value1"}}`,
		jsonStr,
	)

	// ==================
	// Scenario 3
	// ==================
	// And error message resolving also works as expected

	// ---- WHEN
	json, err = fault.ToNaturalJSON("", kt_errors.AllowNonPublicSerialization, kt_errors.ResolveMessages)
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr = string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message with var=value1 and unknown {unknown_var}","isRetryable":true,"errorCodes":["config_error"],"labels":{}}`,
		jsonStr,
	)
}

func TestPublicFaultNaturalJSONSerialization(t *testing.T) {

	// ---- GIVEN

	// we create a Fault - non-public but this does not matter now much
	faultBuilder := kt_errors.NewPublicFaultBuilder(kt_errors.IllegalStateFault).
		WithIsRetryable(true).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithMessageTemplateForAudience("operator", "message for operators var={var1}").
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR).
		WithLabel("var1", "value1").
		WithSource("mymodule", "myfunction")
	fault := faultBuilder.Build()

	// ==================
	// Scenario 1
	// ==================
	// We should see all details - we request the default message

	// ---- WHEN
	json, err := fault.ToNaturalJSON("")
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr := string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message with var={var1} and unknown {unknown_var}","isRetryable":true,"errorCodes":["config_error"],"labels":{"var1":"value1"}}`,
		jsonStr,
	)

	// ==================
	// Scenario 2
	// ==================
	// And error message resolving also works - and this should by default remove labels which were used in the messages as vars as they are not needed being a
	// label anymore.

	// ---- WHEN
	json, err = fault.ToNaturalJSON("", kt_errors.ResolveMessages)
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr = string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message with var=value1 and unknown {unknown_var}","isRetryable":true,"errorCodes":["config_error"],"labels":{}}`,
		jsonStr,
	)

	// ==================
	// Scenario 2b
	// ==================
	// But if we specify 'LeaveMessageVarsInLabels' opt label stays.

	// ---- WHEN
	json, err = fault.ToNaturalJSON("", kt_errors.ResolveMessages, kt_errors.LeaveMessageVarsInLabels)
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr = string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message with var=value1 and unknown {unknown_var}","isRetryable":true,"errorCodes":["config_error"],"labels":{"var1":"value1"}}`,
		jsonStr,
	)

	// ==================
	// Scenario 3
	// ==================
	// If we request not existing audience empty message is returned

	// ---- WHEN
	json, err = fault.ToNaturalJSON("unknown-audience", kt_errors.ResolveMessages)
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr = string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"","isRetryable":true,"errorCodes":["config_error"],"labels":{"var1":"value1"}}`,
		jsonStr,
	)

	// ==================
	// Scenario 4
	// ==================
	// If we request "operator" message that is returned - and again, {var1} label removed

	// ---- WHEN
	json, err = fault.ToNaturalJSON("operator", kt_errors.ResolveMessages)
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr = string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message for operators var=value1","isRetryable":true,"errorCodes":["config_error"],"labels":{}}`,
		jsonStr,
	)

	// ==================
	// Scenario 4b
	// ==================
	// But if we specify 'LeaveMessageVarsInLabels' opt then label stays.

	// ---- WHEN
	json, err = fault.ToNaturalJSON("operator", kt_errors.ResolveMessages, kt_errors.LeaveMessageVarsInLabels)
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr = string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message for operators var=value1","isRetryable":true,"errorCodes":["config_error"],"labels":{"var1":"value1"}}`,
		jsonStr,
	)

}

func TestPublicFaultFullJSONSerialization(t *testing.T) {

	// ---- GIVEN

	// we create a Fault - non-public but this does not matter now much
	faultBuilder := kt_errors.NewPublicFaultBuilder(kt_errors.IllegalStateFault).
		WithIsRetryable(true).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		WithMessageTemplateForAudience("operator", "message for operators var={var2}").
		WithErrorCodes(kt_errors.ILLEGALSTATE_ERRCODE_CONFIG_ERROR).
		WithLabel("var1", "value1").
		WithLabel("var2", "value2").
		WithLabel("var3", "value3").
		WithSource("mymodule", "myfunction")
	fault := faultBuilder.Build()
	controlFault := faultBuilder.Build()
	assert.Equal(t, fault, controlFault)

	// ==================
	// Scenario 1
	// ==================
	// We should see all details

	// ---- WHEN
	json, err := fault.ToFullJSON()
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr := string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message with var={var1} and unknown {unknown_var}","messagesByAudience":{"operator":"message for operators var={var2}"},"isRetryable":true,"errorCodes":["config_error"],"labels":{"var1":"value1","var2":"value2","var3":"value3"}}`,
		jsonStr,
	)

	// ==================
	// Scenario 2
	// ==================
	// We should see all details and resolving also works (this removes labels too)

	// ---- WHEN
	json, err = fault.ToFullJSON(kt_errors.ResolveMessages)
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr = string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message with var=value1 and unknown {unknown_var}","messagesByAudience":{"operator":"message for operators var=value2"},"isRetryable":true,"errorCodes":["config_error"],"labels":{"var3":"value3"}}`,
		jsonStr,
	)
	// original fault should have not been modified anyhow!
	assert.Equal(t, controlFault, fault)

	// ==================
	// Scenario 2b
	// ==================
	// Same as 2 but now we tell the method not to leave labels even after resolving

	// ---- WHEN
	json, err = fault.ToFullJSON(kt_errors.ResolveMessages, kt_errors.LeaveMessageVarsInLabels)
	// ---- THEN
	assert.NoError(t, err)
	// we should see all details
	jsonStr = string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message with var=value1 and unknown {unknown_var}","messagesByAudience":{"operator":"message for operators var=value2"},"isRetryable":true,"errorCodes":["config_error"],"labels":{"var1":"value1","var2":"value2","var3":"value3"}}`,
		jsonStr,
	)
	// original fault should have not been modified anyhow!
	assert.Equal(t, controlFault, fault)
}

func TestAbsolutMinimalisticPublicFaultJSONSerialization(t *testing.T) {

	// ---- GIVEN

	fault := kt_errors.NewPublicFaultBuilder(kt_errors.IllegalStateFault).
		WithMessageTemplate("message with var={var1} and unknown {unknown_var}").
		Build()

	// ==================
	// Scenario 1
	// ==================
	// We go to "natural" JSON representation

	// ---- WHEN
	json, err := fault.ToNaturalJSON("")
	// ---- THEN
	assert.NoError(t, err)
	// no details should be exposed
	jsonStr := string(json)
	assert.Equal(
		t,
		`{"kind":"illegal_state","message":"message with var={var1} and unknown {unknown_var}","isRetryable":false,"errorCodes":[],"labels":{}}`,
		jsonStr,
	)

}
