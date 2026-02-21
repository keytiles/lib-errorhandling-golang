package kt_errors

import (
	"slices"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"github.com/keytiles/lib-utils-golang/pkg/kt_utils"
	"google.golang.org/grpc/codes"
)

// Tests the given error if this is a `Fault` or not. If yes then returns true and the converted Fault.
// If not then returns false and nil. If the provided error is nil, then it is NOT a Fault by default.
func IsFault(err error) (bool, Fault) {
	if err != nil {
		ktErr, ok := err.(Fault)
		if ok {
			return true, ktErr
		}
	}
	return false, nil
}

const (
	logLabelsOption        int = 1
	whitelistedKindsOption int = 2
)

// Can be used as possible option passed into the conversion. Please see methods `OptionXXX()` for supported options!
type ConversionOption interface {
	getOptionId() int
	getLogLabels() []kt_logging.Label
	getKinds() []FaultKind
	getFlag() bool
}

// Conversion option to carry extra log labels.
type optionLogLabels struct {
	logLabels []kt_logging.Label
}

func (o optionLogLabels) getOptionId() int {
	return logLabelsOption
}
func (o optionLogLabels) getLogLabels() []kt_logging.Label {
	return o.logLabels
}
func (o optionLogLabels) getKinds() []FaultKind {
	return nil
}
func (o optionLogLabels) getFlag() bool {
	return false
}

type optionWhiteListedKinds struct {
	kinds             []FaultKind
	inheritErrorCodes bool
}

func (o optionWhiteListedKinds) getOptionId() int {
	return whitelistedKindsOption
}
func (o optionWhiteListedKinds) getLogLabels() []kt_logging.Label {
	return nil
}
func (o optionWhiteListedKinds) getKinds() []FaultKind {
	return o.kinds
}
func (o optionWhiteListedKinds) getFlag() bool {
	return o.inheritErrorCodes
}

// You can pass in labels with this option which will decorate the log event.
//
// But **please note:** if you passed in `transactionId` then it is always added to the log labels. So only for this you do not need to bother with it.
func OptionLogLabels(labels []kt_logging.Label) ConversionOption {
	return optionLogLabels{
		logLabels: labels,
	}
}

// When conversion is made from non-public `Fault` then by default the kind of the converted `Fault` is always `RuntimeFault`. However it is often practical to
// allow a few specific kinds of the `Fault` to be inherited as kind into the public `Fault` during the conversion - instead of hiding those kinds entirely.
//
// With this option you can specify a set of `FaultKind`s to safely inherit.
// In this case the `ERRCODE_INTERNAL_ERROR` is not added if the kind of the original Fault was whitelisted. (As the whitelist itself already suggests a special
// scenario.)
func OptionWhitelistedFaultKinds(inheritErrorCodes bool, kinds ...FaultKind) ConversionOption {
	return optionWhiteListedKinds{
		kinds:             kinds,
		inheritErrorCodes: inheritErrorCodes,
	}
}

// Turns any error into a public Fault instance.
//
// In case the error is already isPublic=true `Fault` then it is returned as it is. Piece of cake :-)
//
// In other cases - even if the error is a `Fault` but isPublic=false - the error is treated as unsafe. Details of the error can easily contain
// implementation details (e.g. we use S3 buckets which failed - the message can reveal this fact we use S3 buckets - not good)
//
// Therefore what happens is that
//   - The method will log the original error. (This is why we need a `loggerToUse` param - see below)
//   - Then construct an isPublic=true `Fault` with generic safe message like "something has happened - details in the log".
//   - Adds error code `ERRCODE_INTERNAL_ERROR`. (If you used `OptionWhitelistedFaultKinds()` that can fine grain this - see description!)
//   - Sets the `cause` of the error to the original error.
//
// In case the original error is isPublic=false `Fault` then we can keep some data from the original error for sure - but with care!
// Retry behavior is alwqys inherited. However the message of the error is still considered unsafe. But if it carries message for audience `MSGAUDIENCE_USER`
// then that one turns into the main message of the converted public error. All labels removed but the ones used in any `messageTemplatesByAudience`. And
// original error codes are also removed. They can potentially again leak out internal implementation details.
//
// Arguments:
//   - 'original': The error you want to turn into a public `Fault`.
//   - 'transactionId': If you have a transaction ID pass it here! Then it will appear in the log as label, added to the Fault as "transactionId" label
//     and also might appear in converted error message. Otherwise pass empty string simply.
//   - 'loggerToUse': This logger is used to log the original fault so we have it, because the converted fault will very likely remove MANY specific
//     details. In case no logger provided then a default Logger will be used for this.
//   - 'options': You can pass in options to the conversion to fine grain how it behaves - please check `kt_errors.OptionXXX()` methods to see possibilities!
func NewPublicFaultFromAnyError(original error, transactionId string, loggerToUse *kt_logging.Logger, options ...ConversionOption) Fault {
	if original == nil {
		return nil
	}
	isFault, fault := IsFault(original)
	if isFault {
		// so the original error is at least a Fault - good!
		if fault.IsPublic() {
			// this is easy - as this is already a public error
			return fault
		}
	}

	var logLabels []kt_logging.Label
	var safeKinds []FaultKind
	kindWasKept := false
	inheritErrorCodes := false
	for _, opt := range options {
		if opt.getOptionId() == logLabelsOption {
			logLabels = opt.getLogLabels()
		} else if opt.getOptionId() == whitelistedKindsOption {
			safeKinds = opt.getKinds()
			inheritErrorCodes = opt.getFlag()
		}
	}

	kind := RuntimeFault
	if isFault && slices.Contains(safeKinds, fault.GetKind()) {
		kind = fault.GetKind()
		kindWasKept = true
	}
	builder := NewPublicFaultBuilder(kind).
		WithCause(original)

	// If we did not keep the original kind mark it as INTERNAL error
	if !kindWasKept {
		builder.WithErrorCodes(ERRCODE_INTERNAL_ERROR)
	}
	if inheritErrorCodes {
		builder.WithErrorCodes(fault.GetErrorCodes()...)
	}

	// let's set a default message
	if transactionId != "" {
		builder.WithMessageTemplate("Error occured during processing, details are logged with transactionId '{transactionId}'").
			WithLabel("transactionId", transactionId)
	} else {
		builder.WithMessageTemplate("Error occured during processing, details are logged")
	}

	// make sure we have a logger - we will need it
	logger := loggerToUse
	if logger == nil {
		logger = getDefaultLogger()
	}
	logEvent := logger.WithLabels(logLabels)

	// is transactionId in labels?
	if transactionId != "" && !slices.ContainsFunc(logLabels, func(item kt_logging.Label) bool { return item.GetStringValue() == transactionId }) {
		// let's enforce we will really decorate the log event with the transaction id!
		logEvent = logEvent.WithLabel(kt_logging.StringLabel("trId", transactionId))
	}
	if isFault {
		logEvent.
			Warn(
				"Unsafe error captured which we turn into a public Fault (kindKept: %t, inheritErrorCodes: %t) - hiding unsafe details. Orig error was: %s",
				kindWasKept, inheritErrorCodes, kt_utils.VarPrinter{TheVar: fault},
			)
		// we can inherit the retry calssification for sure
		builder.WithIsRetryable(fault.IsRetryable())
		audienceMsgTemplates := fault.GetMessageTemplatesByAudience()
		userMsgTemplate := audienceMsgTemplates[MSGAUDIENCE_USER]
		if len(userMsgTemplate) > 0 {
			// the error has a user facing message - let's use this as error message!
			builder.WithMessageTemplate(userMsgTemplate)
			// and remove this from the msg templates
			delete(audienceMsgTemplates, MSGAUDIENCE_USER)
			// and we don't need the transactionId label either
			//builder.WithoutLabels("transactionId")
		}
		// inherit the remaining audience messages into the new error
		builder.WithMessageTemplatesByAudience(audienceMsgTemplates)
		// we keep those labels which are required to resolve any of these messages - but only those
		neededVariables := kt_utils.StringExtractVariableNames(userMsgTemplate)
		for _, audienceMsgTemplate := range audienceMsgTemplates {
			neededVariables.Union(kt_utils.StringExtractVariableNames(audienceMsgTemplate))
		}
		for key, value := range fault.GetLabels() {
			if neededVariables.Contains(key) {
				builder.WithLabel(key, value)
			}
		}
	} else {
		logEvent.
			Warn("Unsafe error captured which we turn into a public Fault - hiding unsafe details. Orig error was: %s",
				kt_utils.VarPrinter{TheVar: original},
			)
	}

	return builder.Build()
}

// Returns the gRPC status code you should use in the error response for the given `Fault`.
//
// IMPORTANT! In case the `Fault` is not public then it is always INTERNAL error - otherwise it is determined from the attributes and the kind of the Fault.
//
// Note: there is an alias for this method as `fault.GetGrpcStatusCode()` - if you prefer that style more.
func GetGrpcStatusCodeForFault(fault Fault) (grpcStatus codes.Code) {
	if fault == nil {
		grpcStatus = codes.OK
		return
	}

	grpcStatus = codes.Internal
	if !fault.IsPublic() {
		return
	}

	// Now lets be error type specific from this point
	switch fault.GetKind() {
	case AuthenticationFault:
		grpcStatus = codes.Unauthenticated
	case AuthorizationFault:
		grpcStatus = codes.PermissionDenied
	case ResourceNotFoundFault:
		grpcStatus = codes.NotFound
	case ConstraintViolationFault:
		grpcStatus = codes.FailedPrecondition
		if fault.HasErrorCode(CONSTRAINTVIOLATION_ERRCODE_ID_ALREADY_TAKEN) ||
			fault.HasErrorCode(CONSTRAINTVIOLATION_ERRCODE_ALREADY_EXIST) {
			grpcStatus = codes.AlreadyExists
		} else if fault.HasErrorCode(CONSTRAINTVIOLATION_ERRCODE_DOES_NOT_EXIST) {
			grpcStatus = codes.NotFound
		}
	case ValidationFault:
		grpcStatus = codes.InvalidArgument
	case NotImplementedFault:
		grpcStatus = codes.Unimplemented
	case IllegalStateFault:
		if fault.HasErrorCode(ILLEGALSTATE_ERRCODE_DEPENDENCY_UNAVAILABLE) ||
			fault.HasErrorCode(ILLEGALSTATE_ERRCODE_TIMED_OUT) {
			grpcStatus = codes.Unavailable
		} else if fault.HasErrorCode(ILLEGALSTATE_ERRCODE_EXHAUSTED) {
			grpcStatus = codes.ResourceExhausted
		} else if fault.HasErrorCode(ILLEGALSTATE_ERRCODE_EXCPECTATION_FAILED) {
			grpcStatus = codes.FailedPrecondition
		}
	}

	return
}

// Returns the HTTP status code you should use in the error response for the given `Fault`.
//
// IMPORTANT! In case the `Fault` is not public then it is always 500 INTERNAL ERROR - otherwise it is determined from the attributes and the kind of the Fault.
//
// Note: there is an alias for this method as `fault.GetHttpStatusCode()` - if you prefer that style more.
func GetHttpStatusCodeForFault(fault Fault) (httpStatus int) {
	if fault == nil {
		httpStatus = 200
		return
	}

	httpStatus = 500
	if !fault.IsPublic() {
		return
	}

	// Now lets be error type specific from this point
	switch fault.GetKind() {
	case AuthenticationFault:
		// UNATHORIZED
		httpStatus = 401
	case AuthorizationFault:
		// FORBIDDEN
		httpStatus = 403
	case ResourceNotFoundFault:
		// NOT_FOUND
		httpStatus = 404
	case ConstraintViolationFault:
		// PRECONDITION_FAILED
		httpStatus = 412
		if fault.HasErrorCode(CONSTRAINTVIOLATION_ERRCODE_ID_ALREADY_TAKEN) ||
			fault.HasErrorCode(CONSTRAINTVIOLATION_ERRCODE_ALREADY_EXIST) {
			// CONFLICT
			httpStatus = 409
		} else if fault.HasErrorCode(CONSTRAINTVIOLATION_ERRCODE_DOES_NOT_EXIST) {
			// NOT_FOUND
			httpStatus = 404
		}
	case ValidationFault:
		// BAD REQUEST
		httpStatus = 400
	case NotImplementedFault:
		// NOT IMPLEMENTED
		httpStatus = 501
	case IllegalStateFault:
		if fault.HasErrorCode(ILLEGALSTATE_ERRCODE_DEPENDENCY_UNAVAILABLE) || fault.HasErrorCode(ILLEGALSTATE_ERRCODE_EXHAUSTED) ||
			fault.HasErrorCode(ILLEGALSTATE_ERRCODE_TIMED_OUT) {
			// SERVICE_UNAVAILABLE
			httpStatus = 503
		} else if fault.HasErrorCode(ILLEGALSTATE_ERRCODE_EXCPECTATION_FAILED) {
			// PRECONDITION_FAILED
			httpStatus = 412
		}
	}

	return
}

// Alias over the Fault's member function `fault.ToNaturalJSON()` - see description there!
// You can also use the member function if you prefer that style more in your code.
func GetFaultAsNaturalJSON(fault Fault, forAudience string, options ...SerializationOption) ([]byte, error) {
	return fault.ToNaturalJSON(forAudience, options...)
}

// Alias over the Fault's member function `fault.ToFullJSON()` - see description there!
// You can also use the member function if you prefer that style more in your code.
func GetFaultAsFullJSON(fault Fault, options ...SerializationOption) ([]byte, error) {
	return fault.ToFullJSON(options...)
}

func getDefaultLogger() *kt_logging.Logger {
	return kt_logging.GetLogger("keytiles.errorhandling")
}
