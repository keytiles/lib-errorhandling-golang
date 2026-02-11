package kt_errors

import (
	"slices"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"github.com/keytiles/lib-utils-golang/pkg/kt_utils"
	"google.golang.org/grpc/codes"
)

var (
	NO_LOG_LABELS []kt_logging.Label = []kt_logging.Label{}
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

// Turns any error into a public Fault instance.
//
// In case the error is already isPublic=true `Fault` then it is returned as it is. Piece of cake :-)
//
// In other cases - even if the error is a `Fault` but isPublic=false - the error is treated as unsafe. Details of the error can easily contain
// implementation details (e.g. we use S3 buckets which failed - the message can reveal this fact we use S3 buckets - not good)
//
// Therefore what happens is that
//   - The method will log the original error.
//   - Then construct an isPublic=true `Fault` with generic safe message like "something has happened - details in the log".
//   - Adds error code `ERRCODE_INTERNAL_ERROR`.
//   - Sets the `cause` of the error to the original error.
//
// In case the original error is isPublic=false `Fault` then we can keep some data from the original error for sure - but with care!
// The message of the error is still considered unsafe. But if it carries message for audience `MSGAUDIENCE_USER` then that one turns into
// the main message of the converted public error. All labels removed but the ones used in any `messageTemplatesByAudience`. Retry behavior
// is inherited. And original error codes are also removed. They can potentially again leak out internal implementation details.
//
// Arguments:
//   - 'original': The error you want to turn into a public `Fault`.
//   - 'transactionId': If you have a transaction ID pass it here! Then it will appear in the log as label, added to the Fault as "transactionId" label
//     and also might appear in converted error message. Otherwise pass empty string simply.
//   - 'loggerToUse': This logger is used to log the original exception so we have it, because the converted error message will not give ANY specific
//     details back. In case no logger provided then a default Logger will be used for this.
//   - 'logLabels': You can pass in labels here which will decorate the log event - or if you dont have / want, you can simply pass in the constant
//     `kt_errors.NO_LOG_LABELS`. **Please note:** if you passed in `transactionId` then it is always added to
//     the log labels! So only for this you have nothing to do here, you can keep this empty.
func NewPublicFaultFromAnyError(original error, transactionId string, loggerToUse *kt_logging.Logger, logLabels []kt_logging.Label) Fault {
	if original == nil {
		return nil
	}
	isFault, Fault := IsFault(original)
	if isFault {
		// so the original error is at least a Fault - good!
		if Fault.IsPublic() {
			// this is easy - as this is already a public error
			return Fault
		}
	}

	builder := NewPublicFaultBuilder(RuntimeFault).
		WithErrorCodes(ERRCODE_INTERNAL_ERROR).
		WithCause(original)
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
				"Unsafe error captured which we turn into a public Fault - hiding unsafe details. Orig error was: %s",
				kt_utils.VarPrinter{TheVar: Fault},
			)
		// we can inherit the retry calssification for sure
		builder.WithIsRetryable(Fault.IsRetryable())
		audienceMsgTemplates := Fault.GetMessageTemplatesByAudience()
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
		for key, value := range Fault.GetLabels() {
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
