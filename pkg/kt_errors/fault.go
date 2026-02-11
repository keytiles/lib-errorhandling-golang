package kt_errors

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/keytiles/lib-sets-golang/ktsets"
	"github.com/keytiles/lib-utils-golang/pkg/kt_utils"
	"google.golang.org/grpc/codes"
)

type FaultKind = string

const (
	// The most generic type without telling too much about the kind of the error - something bad has happened runtime.
	RuntimeFault FaultKind = "runtime"
	// Use this type if your code find itself in an unexpected state.
	// See the predefined `ILLEGALSTATE_ERRCODE_*` constants you can use as error codes to fine grain it further.
	IllegalStateFault FaultKind = "illegal_state"
	// Use this type if something is not implemented.
	NotImplementedFault FaultKind = "not_implemented"
	// You received an input/resource/data but that is not fully valid.
	// See the predefined `VALIDATION_ERRCODE_*` constants you can use as error codes to fine grain it further.
	ValidationFault FaultKind = "validation"
	// You (or user) assumed a certain state but it seems your assumption does not stand...
	// See the predefined `CONSTRAINTVIOLATION_ERRCODE_*` constants you can use as error codes to fine grain it further.
	ConstraintViolationFault FaultKind = "constraint_violation"
	// The resource which expected to be there is actually not. You can also model this with `ConstraintViolationError`
	// combined with `CONSTRAINTVIOLATION_ERRCODE_DOES_NOT_EXIST` error code thats kinda equivalent. But using this might
	// be simpler / more straightforward and this is pretty common problem often.
	ResourceNotFoundFault FaultKind = "resource_not_found"
	// Use this if there was a problem with authentication data.
	// See the predefined `AUTHENTICATION_ERRCODE_*` constants you can use as error codes to fine grain it further.
	AuthenticationFault FaultKind = "authentication"
	// The actor simply does not have permission or we could not determine if he/she/it has.
	// See the predefined `AUTHORIZATION_ERRCODE_*` constants you can use as error codes to fine grain it further.
	AuthorizationFault FaultKind = "authorization"
)

const (
	// This is pretty generic - any case something internally failed we want to mark it like that
	ERRCODE_INTERNAL_ERROR = "internal"

	// The config of the service is somehow wrong and this is causing a bad state
	ILLEGALSTATE_ERRCODE_CONFIG_ERROR = "config_error"
	// A dependency is permanently missing
	ILLEGALSTATE_ERRCODE_DEPENDENCY_MISSING = "missing_dependency"
	// Can be a temporary problem when e.g. we rely on an external system but somehow we can not reach it right now
	ILLEGALSTATE_ERRCODE_DEPENDENCY_UNAVAILABLE = "unavailable_dependency"
	// What we expected did not happen / we got something else
	ILLEGALSTATE_ERRCODE_EXCPECTATION_FAILED = "expectation_failed"
	// Something timed out - job is not done, state is not good
	ILLEGALSTATE_ERRCODE_TIMED_OUT = "timed_out"
	// Something has reached its limits - no more is possible
	ILLEGALSTATE_ERRCODE_EXHAUSTED = "exhausted"
	// We tried to serialize something into JSON/Yaml/binary etc but it failed. This often can indicate a problem with the original input.
	ILLEGALSTATE_ERRCODE_SERIALIZATION_FAILED = "serialization_failed"
	// We tried to deserialize something from JSON/Yaml/binary etc but it failed. This often can indicate a problem with the original input.
	ILLEGALSTATE_ERRCODE_DESERIALIZATION_FAILED = "deserialization_failed"
	// You can use this if you think this error only possible if we clearly have a bug in the code.
	// Time to time happens you find yourself in an error handling case you know "this is impossible" if I find myself here.
	ILLEGALSTATE_ERRCODE_CODE_BUG = "code_bug"

	// Use this error code if you expected something else as a type
	VALIDATION_ERRCODE_WRONG_DATATYPE = "wrong_datatype"
	// Use this error code if you expected a sepcific format for something but you received something else instead
	VALIDATION_ERRCODE_WRONG_FORMAT = "wrong_format"
	// Use this error code if you expected a mandatory parameter but was not provided
	VALIDATION_ERRCODE_MISSING_MANDATORY = "mandatoy_info_missing"
	// Use this error code if somewhere you expected to get nothing (e.g. a field should be None) but you got something
	VALIDATION_ERRCODE_SHOULD_NOT_BE_PROVIDED = "should_not_be_provided"
	// Use this error code if however data was provided it is not valid - content wise
	VALIDATION_ERRCODE_INVALID_VALUE = "invalid_value"
	// Use this error code if provided data is trying to change a value which actually is read-only
	VALIDATION_ERRCODE_READONLY_VALUE_CHANGED = "readonly_value_changed"

	// Use this error code if you have a resource conflicting PrimaryKey or ID
	CONSTRAINTVIOLATION_ERRCODE_ID_ALREADY_TAKEN = "id_already_taken"
	// A bit more generic representation of the fact: something already exists
	CONSTRAINTVIOLATION_ERRCODE_ALREADY_EXIST = "already_exists"
	// The object / resource / whatever we were expected being there is actually not there
	CONSTRAINTVIOLATION_ERRCODE_DOES_NOT_EXIST = "not_exists"
	// A generic description of the fact that the preconditions you were expected is not met
	CONSTRAINTVIOLATION_ERRCODE_PRECONDITION_FAILED = "precondition_failed"
	// Use this error code if you have a resource conflicting assumed vs real versions
	CONSTRAINTVIOLATION_ERRCODE_VERSION_CONFLICT = "resource_version_conflict"

	// Use this if you expected to have an authentication at certain point but it is not there
	AUTHENTICATION_ERRCODE_MISSING = "auth_data_missing"
	// Use this if however auth info was there but it is using a method which you do not support
	AUTHENTICATION_ERRCODE_NOT_SUPPORTED = "auth_method_not_supported"
	// Use this if however auth info was there auth process was not successful
	AUTHENTICATION_ERRCODE_FAILED = "authentication_failed"

	// The actor can not do this.
	AUTHORIZATION_NO_PERMISSION = "no_permission"
	// Use this if the authorization process was not successful for whatever reason. So this does not mean
	// the actor has no permission, it just failed this time.
	AUTHORIZATION_ERRCODE_FAILED = "authorization_failed"
)

const (
	// Audience role - user - for audience facing message templates.
	MSGAUDIENCE_USER = "user"
)

// Basically true/false options to change the serialization behavior
type SerializationOption int

const (
	// If set then the possible {var} variables are getting resolved from the labels in the messages before returned.
	// And in this case these {var} variables by default also removed from the labels as they became part of the message - unless
	// you pass `LeaveMessageVarsInLabels` option.
	ResolveMessages SerializationOption = 1
	// If `ResolveMessages` is used but you explicitly want to leave the {var} variables in the labels (removed by default as they) use this option.
	LeaveMessageVarsInLabels = 2

	// If set then the JSON is indented with line breaks and tabs so becomes more human readable
	PrettyPrint = 3
	// By default the serialization only happens if Fault is public - to prevent data leak non-public Faults simply returning blank form.
	// But if you set this option explicitly then this defense mechanism gets disabled.
	AllowNonPublicSerialization = 4
)

// Our unified, data rich Keytiles-internal error which is able to carry many and all necessarry information and let it bubble up from literally any layers:
// even from libraries or simply service internal layers.
//
// The most important concept:
// An error **can be classified as "public"** - yes or no.
// Public errors are suitable to leave the boundary and even show it to users - as they are phrased the way a) message is clear b) not leaking out internal
// implementation details for sure.
// If an error is non-Public then it is considered unsafe in the above sense and can be converted into a public version using method
// `NewPublicFaultFromAnyError()` - see method comments to get a better picture what is happening then!
//
// An error like this is data rich - as we already wrote. It can
//   - Carry error codes - for machine readability (strings - see predefined `*_ERRCODE_*` codes, but you can also define your own)
//   - The error is typed - see `ErrorType`s! This also helps a lot in machine readability as well as properly convert then e.g. to Http/gRPC status codes!
//   - Carry the cause - the error which caused this error
//   - Carry labels (key-value pairs) associated with this error
//   - The message is not just a dumb string but can be a template! It can contain variable placeholders
//     (Python style - "My error message with {var1} and {var2}") which can be resolved from labels!
//   - And apart from the default message it can also carry different message templates meant for different audiences
//
// To construct an error with comfort we use builder pattern. You can use `NewFaultBuilder()` or `NewPublicFaultBuilder()` to quickly
// create a builder for a non-public or public error and fine tune the data it will carry.
//
// Please note: this object is semi-immutable! Many things you can NOT change but some info you can extend, add to it (mutate) as error bubbles
// upwards the call chain. See `AddXXX()` methods!
//
// IMPORTANT NOTE - for logging or printing into string:
// Use `VarPrinter` if you want the error printed using its `String()` method! Otherwise the `Error()` method will be used by Go by default as this
// is `error` type. The `Error()` is just returning / printing the message and optionally error codes, labels. Human readable form. Full info printed
// only by `String()` method which is probably what you want to log.
type Fault interface {
	error
	fmt.Stringer

	// Returns the type of this error.
	GetKind() FaultKind
	// Returns the message template unresolved (so with possible variable placeholders in it as is)
	GetMessageTemplate() string
	// Returns the message - with resolved variable placeholders from labels.
	GetMessage() string
	// Returns the message template meant for the given audience unresolved (so with possible variable placeholders in it as is).
	// If there is no template for the requested audience, empty string is returned.
	GetMessageTemplateForAudience(forAudience string) string
	// Returns the message meant for the given audience - with resolved variable placeholders from labels.
	// If there is no template for the requested audience, empty string is returned.
	GetMessageForAudience(forAudience string) string
	// Returns map view of message templates by audiences.
	// **Note:** This always makes and returns a copy so use it accordingly!
	GetMessageTemplatesByAudience() map[string]string
	// Tells if this error is suitable to leave the private boundary or not (public = no implementation details leaking for sure).
	IsPublic() bool
	// We extend the error with the possibility of check if error is retryable.
	IsRetryable() bool
	// Returns all associated error codes.
	// **Note:** This always makes and returns a copy so use it accordingly! If possible use `HasErrorCode()` instead.
	GetErrorCodes() []string
	// Tells if this error is carrying ANY of the listed error codes or not.
	HasErrorCode(codes ...string) bool
	// Returns the Cause of this error - which is another (any) error.
	GetCause() error
	// Errors can carry a set of labels. This returns them all.
	// **Note:** This always makes and returns a copy so use it accordingly! If you can use `GetLabel()` method instead.
	GetLabels() map[string]any
	// Returns a specific label if Fault has it - or Nil if does not have it. You can also take and use the returned `found` flag.
	GetLabel(key string) (value any, found bool)
	// Error supports tracking the call chain. You can optionally use this (or not, up to you). But if you do, this method returns the content of this.
	// The `GetSource()` method returns where the error was born - you can set this with the builder `WithSource()` method. Then as the error bubbles
	// up, each hop can use the `AddCallerToCallStack()` method. This is how call stack is building up - what you can retrieve with this method.
	// The last element is the source - returned by `GetSource()`. Then the previous element is who called the source. And so on. The first element
	// is the point who started the whole call chain.
	GetCallStack() []string
	// Tells you where the error is originated from. We do it the easiest way: we can put this into a string :-) That's it.
	// See the error builder `WithSource()` method! If you invoke `GetCallStack()` method, this will be actually the deepest element on the stack.
	GetSource() string

	// You can add a caller to the call stack. You can do this when you capture an error like this because it is returned to you.
	// As you can see, if you want you can pass in multiple string elements. If you do so, they will be automatically concatenated
	// using "." separator. Why is it useful? Because you can do something like this: `AddCallerToCallStack("mypackage", "mymethod")` e.g.
	AddCallerToCallStack(caller ...string)
	// As the error bubbles upwards in higher layers it is often a requirement you want to add a bit more context to it. In classic error handling this
	// often ends in building mapping-functions and you raise a completely new error attaching the original one as Cause or similar tacticts. However
	// this leads to lots of boilerplate code and often results in mistakes especially if you have multiple layers in your code and each one is doing the same.
	//
	// So instead enforcing this strategy `Fault` offers a simpler way. You do not need to create a brand new error just to add your context info but you
	// can simply extend it in place and let the original prublom bubble upwards with all the details it already has! Basically, you simply extend the
	// information with those details you know in the higher level layer only.
	//
	// Using this method you can do this. You can prepend (prefix) to the messageTemplate of the error with a piece of string.
	// It is really a prefix - imagine a simple concatenation! So you need to include separators, white-spaces etc at the end of your prefix str!
	// If you send in empty str nothing will happen.
	AddContextToMessage(msgTemplatePrefix string)
	// Same as `AddContextToMessage()` (read its comment!) but with this one you can extend the audience facing messages with more context. If the audience you
	// refer to with `forAudience` does not exist it will be created. And maybe good to know that the `msgTemplatePrefix` value in this case will be trimmed on
	// the right side (not just whitespaces but also ':' and '-' characters) so no need to worry about strange white spaces.
	// If you send in empty str in any parameters nothing will happen.
	AddContextToAudienceMessage(forAudience string, msgTemplatePrefix string)
	// Please read the comment of `AddContextToMessage()` method! You get a better understanding on the motivation and problem then.
	// With this method - as the error bubbles upwards - highler level layers might want to extend it with their custom error codes. You can do it in one go by
	// adding multiple at once.
	AddErrorCodes(c ...string)
	// Please read the comment of `AddContextToMessage()` method! You get a better understanding on the motivation and problem then.
	// As the error bubbles upwards higher level layers might want to extend it with more labels - especially since we have `AddContextToMessage()` and
	// `AddContextToAudienceMessage()` which can introduce new {var}-s into the messages.
	AddLabel(key string, value any)
	// Please read the comment of `AddContextToMessage()` method! You get a better understanding on the motivation and problem then.
	// As the error bubbles upwards higher level layers might want to extend it with more labels - especially since we have `AddContextToMessage()` and
	// `AddContextToAudienceMessage()` which can introduce new {var}-s into the messages.
	AddLabels(labels map[string]any)

	// Returns the HTTP status code you should use in the response if you fail from this Fault.
	// Note: this is a wrapper around the utility function `GetHttpStatusCodeForFault()` - you can use that if you prefer that form instead.
	// IMPORTANT! In case the `Fault` is not public then it is always 500 INTERNAL ERROR - otherwise it is determined from the attributes and the kind of the
	// Fault.
	GetHttpStatusCode() int
	// Returns the gRPC status code you should use in the response if you fail from this Fault.
	// Note: this is a wrapper around the utility function `GetGrpcStatusCodeForFault()` - you can use that if you prefer that form instead.
	// IMPORTANT! In case the `Fault` is not public then it is always INTERNAL error - otherwise it is determined from the attributes and the kind of the Fault.
	GetGrpcStatusCode() codes.Code

	// Returns the natural (most human readable) JSON form of this Fault - can come handy if you build e.g. HTTP APIs and you need quickly return an error
	// response. Check the available `SerializationOption`s you can use optionally!
	// This method returns a JSON like:
	//
	//    {
	//       "kind": "<the Kind>",
	//       "message": "<the default MessageTemplate or given 'forAudience' template - raw or resolved>",
	//       "isRetryable": true/false,
	//       "errorCodes": ["the", "error", "codes"],
	//       "labels": {
	//           "key1": <value1>,
	//           "key2": <value2>,
	//           ...
	//        }
	//    }
	//
	// As you see really internal details like "cause" or "call stack" etc are absolutely not revealed.
	//
	// IMPORTANT! To prevent accidental data leak this serialization only renders public Faults! If the Fault is non-public you get back empty
	// values only - unless you explicitly use `AllowNonPublicSerialization` option!
	//
	// Parameters:
	// - `forAudience` - if you pass empty string you get back the default MessageTemplate - otherwise the specific audience message comes back
	ToNaturalJSON(forAudience string, options ...SerializationOption) ([]byte, error)

	// Just like `ToNaturalJSON()` this also returns a JSON representation but this one returns the "message" and "messagesByAudience"
	// separately - revealing more internal structure.
	//
	// However really internal details like "cause" or "call stack" etc are absolutely not revealed even in this form.
	//
	// IMPORTANT! To prevent accidental data leak this serialization only renders public Faults! If the Fault is non-public you get back empty
	// values only - unless you explicitly use `AllowNonPublicSerialization` option!
	ToFullJSON(options ...SerializationOption) ([]byte, error)
}

func newInitializedFault(errType FaultKind) defaultFault {
	return defaultFault{
		Kind: errType,
		// lets keep these on Nil until first used
		//Labels:                     make(map[string]any, 0),
		//MessageTemplatesByAudience: make(map[string]string, 0),
		callStack: make([]string, 0, 4),
	}
}

// This is used only for JSON / Yaml serialization
type naturalFormFault struct {
	Kind       FaultKind      `json:"kind" yaml:"kind"`
	Message    string         `json:"message" yaml:"message"`
	Retryable  bool           `json:"isRetryable" yaml:"isRetryable"`
	ErrorCodes []string       `json:"errorCodes" yaml:"errorCodes"`
	Labels     map[string]any `json:"labels" yaml:"labels"`
}

type defaultFault struct {
	Kind                       FaultKind         `json:"kind" yaml:"kind"`
	MessageTemplate            string            `json:"message" yaml:"message"`
	MessageTemplatesByAudience map[string]string `json:"messagesByAudience" yaml:"messagesByAudience"`
	Retryable                  bool              `json:"isRetryable" yaml:"isRetryable"`
	ErrorCodes                 []string          `json:"errorCodes" yaml:"errorCodes"`
	Labels                     map[string]any    `json:"labels" yaml:"labels"`
	properties                 map[string]any
	public                     bool
	cause                      error
	callStack                  []string
}

func (fault *defaultFault) GetKind() FaultKind {
	if fault == nil {
		return ""
	}
	return fault.Kind
}

func (fault *defaultFault) GetMessageTemplate() string {
	if fault == nil {
		return ""
	}
	return fault.MessageTemplate
}

func (fault *defaultFault) GetMessage() string {
	if fault == nil {
		return ""
	}
	return kt_utils.StringSimpleResolve(fault.MessageTemplate, fault.Labels)
}

func (fault *defaultFault) GetMessageTemplateForAudience(forAudience string) string {
	if fault == nil || fault.MessageTemplatesByAudience == nil {
		return ""
	}
	return fault.MessageTemplatesByAudience[forAudience]
}

func (fault *defaultFault) GetMessageForAudience(forAudience string) string {
	if fault == nil || fault.MessageTemplatesByAudience == nil {
		return ""
	}
	return kt_utils.StringSimpleResolve(fault.GetMessageTemplateForAudience(forAudience), fault.Labels)
}

func (fault *defaultFault) GetMessageTemplatesByAudience() map[string]string {
	if fault == nil || fault.MessageTemplatesByAudience == nil {
		return make(map[string]string)
	}
	// we return a copy only
	ret := make(map[string]string, len(fault.MessageTemplatesByAudience))
	maps.Copy(ret, fault.MessageTemplatesByAudience)
	return ret
}

func (fault *defaultFault) IsPublic() bool {
	if fault == nil {
		return false
	}
	return fault.public
}

func (fault *defaultFault) GetErrorCodes() []string {
	if fault == nil || fault.ErrorCodes == nil {
		// we return empty
		return make([]string, 0)
	}
	// we return a copy
	ret := make([]string, len(fault.ErrorCodes))
	copy(ret, fault.ErrorCodes)
	return ret
}

func (fault *defaultFault) HasErrorCode(codes ...string) bool {
	if fault == nil || fault.ErrorCodes == nil {
		return false
	}
	for _, code := range codes {
		if slices.Contains(fault.ErrorCodes, code) {
			return true
		}
	}
	return false
}

func (fault *defaultFault) GetCause() error {
	if fault == nil {
		return nil
	}
	return fault.cause
}

func (fault *defaultFault) GetSource() string {
	if fault == nil {
		return ""
	}
	if len(fault.callStack) > 0 {
		return fault.callStack[0]
	}
	return ""
}

func (fault *defaultFault) GetCallStack() []string {
	if fault == nil {
		return make([]string, 0)
	}
	// we return a copy
	ret := make([]string, len(fault.callStack))
	copy(ret, fault.callStack)
	slices.Reverse(ret)
	return ret
}

func (fault *defaultFault) AddCallerToCallStack(caller ...string) {
	if fault == nil {
		return
	}
	fault.callStack = append(fault.callStack, strings.Join(caller, "."))
}

func (fault *defaultFault) IsRetryable() bool {
	if fault == nil {
		return false
	}
	return fault.Retryable
}

func (fault *defaultFault) GetLabel(key string) (value any, found bool) {
	if fault == nil || fault.Labels == nil || key == "" {
		return
	}
	value, found = fault.Labels[key]
	return
}

func (fault *defaultFault) GetLabels() map[string]any {
	if fault == nil || fault.Labels == nil {
		// we return empty map
		return make(map[string]any)
	}
	// we return a copy only
	ret := make(map[string]any, len(fault.Labels))
	maps.Copy(ret, fault.Labels)
	return ret
}

func (fault *defaultFault) AddContextToMessage(contextMsgTemplate string) {
	if fault == nil {
		return
	}
	if contextMsgTemplate != "" {
		// we prepend to the message
		fault.MessageTemplate = contextMsgTemplate + fault.MessageTemplate
	}
}

func (fault *defaultFault) AddContextToAudienceMessage(forAudience string, contextMsgTemplate string) {
	if fault == nil {
		return
	}
	if contextMsgTemplate != "" && forAudience != "" {
		_trimmed := ""
		msg, found := fault.MessageTemplatesByAudience[forAudience]
		if found {
			// we prepend to the message
			fault.MessageTemplatesByAudience[forAudience] = contextMsgTemplate + msg
		}
		if !found {
			// will become the message but trimmed way
			if _trimmed == "" {
				// we just do it once
				_trimmed = strings.TrimRight(contextMsgTemplate, " \t\r\n-:")
			}
			fault.MessageTemplatesByAudience[forAudience] = _trimmed
		}

	}
}

func (fault *defaultFault) AddErrorCodes(c ...string) {
	if fault == nil {
		return
	}
	if fault.ErrorCodes == nil {
		fault.ErrorCodes = make([]string, 0, len(c))
	}
	for _, errCode := range c {
		if errCode != "" && !slices.Contains(fault.ErrorCodes, errCode) {
			fault.ErrorCodes = append(fault.ErrorCodes, errCode)
		}
	}
}

func (fault *defaultFault) AddLabel(key string, value any) {
	if fault == nil {
		return
	}
	if key != "" {
		// lets lazy-create map if not created yet
		if fault.Labels == nil {
			fault.Labels = make(map[string]any)
		}
		fault.Labels[key] = value
	}
}

func (fault *defaultFault) AddLabels(labels map[string]any) {
	if fault == nil || labels == nil {
		return
	}
	// lets lazy-create map if not created yet
	if fault.Labels == nil {
		fault.Labels = make(map[string]any, len(labels))
	}
	maps.Copy(fault.Labels, labels)
}

func (fault *defaultFault) GetHttpStatusCode() int {
	return GetHttpStatusCodeForFault(fault)
}

func (fault *defaultFault) GetGrpcStatusCode() codes.Code {
	return GetGrpcStatusCodeForFault(fault)
}

var (
	_EMPTY_NATURAL_FORM = naturalFormFault{
		Kind:       "NaN",
		Message:    "",
		ErrorCodes: make([]string, 0),
		Labels:     make(map[string]any, 0),
		Retryable:  false,
	}

	_NONPUBLIC_NATURAL_FORM = naturalFormFault{
		Kind:       RuntimeFault,
		Message:    "",
		ErrorCodes: make([]string, 0),
		Labels:     make(map[string]any, 0),
		Retryable:  false,
	}

	_EMPTY_FAULT = defaultFault{
		Kind:                       "NaN",
		MessageTemplate:            "",
		MessageTemplatesByAudience: make(map[string]string, 0),
		ErrorCodes:                 make([]string, 0),
		Labels:                     make(map[string]any, 0),
		Retryable:                  false,
	}

	_NONPUBLIC_FAULT = defaultFault{
		Kind:                       "NaN",
		MessageTemplate:            "",
		MessageTemplatesByAudience: make(map[string]string, 0),
		ErrorCodes:                 make([]string, 0),
		Labels:                     make(map[string]any, 0),
		Retryable:                  false,
	}
)

func (fault *defaultFault) ToNaturalJSON(forAudience string, options ...SerializationOption) ([]byte, error) {
	var natural naturalFormFault
	if fault == nil {
		natural = _EMPTY_NATURAL_FORM
	} else if !fault.IsPublic() && !slices.Contains(options, AllowNonPublicSerialization) {
		natural = _NONPUBLIC_NATURAL_FORM
		// this is safe to inherit
		natural.Retryable = fault.Retryable
	} else {
		natural = naturalFormFault{
			Kind:       fault.Kind,
			Retryable:  fault.Retryable,
			ErrorCodes: fault.ErrorCodes,
		}
		if natural.ErrorCodes == nil {
			natural.ErrorCodes = make([]string, 0)
		}

		resolveMessages := slices.Contains(options, ResolveMessages)
		leaveVars := slices.Contains(options, LeaveMessageVarsInLabels)

		if resolveMessages && !leaveVars {
			// we need a copy / empty map
			natural.Labels = fault.GetLabels()
		} else {
			// for sure we will not manipulate labels - we dont need copy
			natural.Labels = fault.Labels
		}
		if natural.Labels == nil {
			natural.Labels = make(map[string]any)
		}

		var msgVars ktsets.Set[string]
		if forAudience == "" {
			if resolveMessages {
				natural.Message = fault.GetMessage()
				if !leaveVars {
					msgVars = kt_utils.StringExtractVariableNames(fault.MessageTemplate)
				}
			} else {
				natural.Message = fault.MessageTemplate
			}
		} else {
			if resolveMessages {
				natural.Message = fault.GetMessageForAudience(forAudience)
				if !leaveVars {
					msgVars = kt_utils.StringExtractVariableNames(fault.MessageTemplatesByAudience[forAudience])
				}
			} else {
				natural.Message = fault.MessageTemplatesByAudience[forAudience]
			}
		}
		if msgVars.Size() > 0 {
			for _, k := range msgVars.GetAll() {
				delete(natural.Labels, k)
			}
		}
	}

	if slices.Contains(options, PrettyPrint) {
		return json.MarshalIndent(natural, "", "\t")
	} else {
		return json.Marshal(natural)
	}
}

func (fault *defaultFault) ToFullJSON(options ...SerializationOption) ([]byte, error) {

	resolveMessages := slices.Contains(options, ResolveMessages)
	leaveVars := slices.Contains(options, LeaveMessageVarsInLabels)

	var _fault defaultFault
	if fault == nil {
		_fault = _EMPTY_FAULT
	} else if !fault.IsPublic() && !slices.Contains(options, AllowNonPublicSerialization) {
		_fault = _NONPUBLIC_FAULT
		// this is safe to inherit
		_fault.Retryable = fault.Retryable
	} else {
		_fault = *fault
		if resolveMessages && !leaveVars {
			// we need a copy of labels - as we might manipulate them and this should not affect original
			_fault.Labels = fault.GetLabels()
		}
	}

	if resolveMessages {
		var msgVars ktsets.Set[string]
		_fault.MessageTemplate = fault.GetMessage()
		if !leaveVars {
			msgVars = kt_utils.StringExtractVariableNames(fault.MessageTemplate)
		}
		// we need to work on a copy before we alter it - to avoid changing original
		_fault.MessageTemplatesByAudience = make(map[string]string, len(fault.MessageTemplatesByAudience))
		for k := range fault.MessageTemplatesByAudience {
			_fault.MessageTemplatesByAudience[k] = fault.GetMessageForAudience(k)
			if !leaveVars {
				msgVars.Union(kt_utils.StringExtractVariableNames(fault.MessageTemplatesByAudience[k]))
			}
		}

		if msgVars.Size() > 0 {
			for _, k := range msgVars.GetAll() {
				delete(_fault.Labels, k)
			}
		}
	}

	if slices.Contains(options, PrettyPrint) {
		return json.MarshalIndent(_fault, "", "\t")
	} else {
		return json.Marshal(_fault)
	}
}

// The implementation of Error iface - this considers if the error is public or not.
// If not public then just prints the resolved message and safe info (to avoid leaking internal info) - otherwise also reveals labels
func (fault *defaultFault) Error() string {
	codesStr := "[]"
	if len(fault.ErrorCodes) > 0 {
		codesStr = fmt.Sprintf("['%s']", strings.Join(fault.ErrorCodes, "','"))
	}
	if fault.public {
		return fmt.Sprintf("%s: %s (retryable: %t, errorCodes: %s, labels: %s)",
			fault.Kind, fault.GetMessage(), fault.Retryable, codesStr, kt_utils.PrintVarS(fault.Labels, false))
	} else {
		return fmt.Sprintf("%s: %s (retryable: %t, errorCodes: %s)",
			fault.Kind, fault.GetMessage(), fault.Retryable, codesStr)
	}
}

// The fmt.Stringer implementation which is producing complete string representation of the error. Useful for logging purposes.
func (fault *defaultFault) String() string {
	causeStr := "nil"
	if fault.cause != nil {
		isKtErr, ktErr := IsFault(fault.cause)
		if isKtErr {
			// we use the to string mechanism
			causeStr = fmt.Sprintf("{%s}", ktErr.String())
		} else {
			// we print it normal way
			causeStr = fmt.Sprintf("'%s'", fault.cause)
		}

	}
	codesStr := "[]"
	if len(fault.ErrorCodes) > 0 {
		codesStr = fmt.Sprintf("['%s']", strings.Join(fault.ErrorCodes, "','"))
	}
	callStackStr := "[]"
	if len(fault.callStack) > 0 {
		callStackStr = fmt.Sprintf("['%s']", strings.Join(fault.GetCallStack(), "','"))
	}
	audMsgsStr := "{}"
	if len(fault.MessageTemplatesByAudience) > 0 {
		audMsgsStr = kt_utils.PrintVarS(fault.MessageTemplatesByAudience, false)
	}
	labStr := "{}"
	if len(fault.Labels) > 0 {
		labStr = kt_utils.PrintVarS(fault.Labels, false)
	}

	return fmt.Sprintf(
		"Fault{type: '%s', msgTemplate: '%s', retryable: %t, public: %t, codes: %s, callStack: %s, cause: %s, audienceMsgs: %s, labels: %s}",
		fault.Kind,
		fault.MessageTemplate,
		fault.Retryable,
		fault.public,
		codesStr,
		callStackStr,
		causeStr,
		audMsgsStr,
		labStr,
	)
}
