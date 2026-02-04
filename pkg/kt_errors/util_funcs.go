package kt_errors

import (
	"slices"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"github.com/keytiles/lib-utils-golang/pkg/kt_utils"
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

func getDefaultLogger() *kt_logging.Logger {
	return kt_logging.GetLogger("keytiles.errorhandling")
}
