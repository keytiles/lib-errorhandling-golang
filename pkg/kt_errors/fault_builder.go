package kt_errors

import (
	"maps"
	"strings"

	"github.com/keytiles/lib-sets-golang/ktsets"
)

// Creates a new FaultBuilder for "public" errors and you can convenient way fine tune the error before you invoke `Build()` method on it.
// You must specify the kind of the error right away - this is not changeable later.
func NewPublicFaultBuilder(errType FaultKind) *FaultBuilder {
	err := newInitializedFault(errType)
	err.public = true
	return &FaultBuilder{fault: err, errCodes: ktsets.NewSet[string]()}
}

// Creates a new FaultBuilder marked "non public" and you can convenient way fine tune the error before you invoke `Build()` method on it.
// You must specify the kind of the error right away - this is not changeable later.
func NewFaultBuilder(errType FaultKind) *FaultBuilder {
	err := newInitializedFault(errType)
	err.public = false
	return &FaultBuilder{fault: err, errCodes: ktsets.NewSet[string]()}
}

type FaultBuilder struct {
	fault    defaultFault
	errCodes ktsets.Set[string]
}

func (builder *FaultBuilder) Build() Fault {
	_fault := builder.fault

	// labels remain mutable - so we need a copy there from the builder
	if builder.fault.Labels != nil {
		_fault.Labels = builder.fault.GetLabels()
	}

	// assemble error codes
	if builder.errCodes.Size() > 0 {
		_fault.ErrorCodes = builder.errCodes.GetAll()
	}

	// review the isRetryable flag
	if builder.fault.Retryable {
		switch builder.fault.Kind {
		case AuthenticationFault:
			if builder.errCodes.ContainsAny(AUTHENTICATION_ERRCODE_MISSING, AUTHENTICATION_ERRCODE_NOT_SUPPORTED) {
				builder.fault.Retryable = false
			}
		case AuthorizationFault:
			if builder.errCodes.ContainsAny(AUTHORIZATION_NO_PERMISSION) {
				builder.fault.Retryable = false
			}
		}
	}

	return &_fault
}

// Sets if this error is retryable or not.
//
// Please note: certain error types are inheritedly not retryable, e.g. ValidationError or NotImplementedError. Invoking this method
// on any of those will simply have no effect.
func (builder *FaultBuilder) WithIsRetryable(flag bool) *FaultBuilder {
	switch builder.fault.Kind {
	// only these types can be classified as retryable
	case NotImplementedFault, ValidationFault, ResourceNotFoundFault:
		// we skip it - these are inheritedly not retryable
	default:
		builder.fault.Retryable = flag
	}
	return builder
}

// Setting the message template of the error.
// Why is it a "template"? Because you can use variables in it (Python style), e.g. "My string with {var1} and {var2} variables.".
// Then add these as labels (see `WithLabel()` / `WithLabels()`).
func (builder *FaultBuilder) WithMessageTemplate(msg string) *FaultBuilder {
	builder.fault.MessageTemplate = msg
	return builder
}

// Sets a message template for a specific audience.
func (builder *FaultBuilder) WithMessageTemplateForAudience(forAudience string, msg string) *FaultBuilder {
	if builder.fault.MessageTemplatesByAudience == nil {
		builder.fault.MessageTemplatesByAudience = make(map[string]string)
	}
	builder.fault.MessageTemplatesByAudience[forAudience] = msg
	return builder
}

// If you changed your mind you can remove the template for this audience
func (builder *FaultBuilder) WithoutMessageTemplateForAudiences(forAudiences ...string) *FaultBuilder {
	if builder.fault.MessageTemplatesByAudience == nil {
		return builder
	}
	for _, audience := range forAudiences {
		delete(builder.fault.MessageTemplatesByAudience, audience)
	}
	if len(builder.fault.MessageTemplatesByAudience) == 0 {
		builder.fault.MessageTemplatesByAudience = nil
	}
	return builder
}

// Adds all audience message templates to the error - this is a merge.
func (builder *FaultBuilder) WithMessageTemplatesByAudience(templates map[string]string) *FaultBuilder {
	if len(templates) == 0 {
		return builder
	}
	if builder.fault.MessageTemplatesByAudience == nil {
		builder.fault.MessageTemplatesByAudience = make(map[string]string, len(templates))
	}
	maps.Copy(builder.fault.MessageTemplatesByAudience, templates)
	return builder
}

// Adds all audience message templates to the error - and these will override the possibly existing ones.
func (builder *FaultBuilder) WithExactMessageTemplatesByAudience(templates map[string]string) *FaultBuilder {
	if len(templates) == 0 {
		builder.fault.MessageTemplatesByAudience = nil
		return builder
	}

	builder.fault.MessageTemplatesByAudience = make(map[string]string, len(templates))
	maps.Copy(builder.fault.MessageTemplatesByAudience, templates)
	return builder
}

// You can attach the error which caused this error to this error.
func (builder *FaultBuilder) WithCause(e error) *FaultBuilder {
	builder.fault.cause = e
	return builder
}

// Removes the attached cause from the error - if there was attached any previously.
func (builder *FaultBuilder) WithoutCause() *FaultBuilder {
	builder.fault.cause = nil
	return builder
}

// You can attach info to the error regarding where is it coming from?
// We do not use stacktraces from runtime as that is expensive and overkill. Yet, it can be helpful to know from where
// the error originates from. We do it in the easiest way: you can put this into a string the way you want :-) That's it.
// As you can see, if you want you can pass in multiple string elements. If you do so, they will be automatically concatenated
// using "." separator.
func (builder *FaultBuilder) WithSource(src ...string) *FaultBuilder {
	builder.fault.callStack = append(builder.fault.callStack, strings.Join(src, "."))
	return builder
}

// You can add error codes to this error - multiple in one call.
// Error codes are simply strings. There are several predefined ones - see `*_ERRCODE_*` constants - but you can also
// define you owns of course.
func (builder *FaultBuilder) WithErrorCodes(c ...string) *FaultBuilder {
	builder.errCodes.AddAll(c...)
	return builder
}

// If you changed your mind you can remove specific error codes from the error.
func (builder *FaultBuilder) WithoutErrorCodes(c ...string) *FaultBuilder {
	builder.errCodes.RemoveAll(c...)
	return builder
}

// Attaching a label (key-value pair) to this error.
func (builder *FaultBuilder) WithLabel(key string, value any) *FaultBuilder {
	builder.fault.AddLabel(key, value)
	return builder
}

// If you changed your mind you can remove specific labels (key-value pair) from this error.
func (builder *FaultBuilder) WithoutLabels(keys ...string) *FaultBuilder {
	if builder.fault.Labels == nil {
		return builder
	}
	for _, key := range keys {
		delete(builder.fault.Labels, key)
	}
	if len(builder.fault.Labels) == 0 {
		builder.fault.Labels = nil
	}
	return builder
}

// You can attach multiple labels (key-value pairs) in one go if you wish with this method.
// Please note that these will be simply merged into the existing labels! See also `WithExactLabels()` method!
func (builder *FaultBuilder) WithLabels(labels map[string]any) *FaultBuilder {
	builder.fault.AddLabels(labels)
	return builder
}

// Sets the labels (key-value pairs) attached to this error to the given map - all previous labels will be removed.
func (builder *FaultBuilder) WithExactLabels(labels map[string]any) *FaultBuilder {
	if len(labels) == 0 {
		builder.fault.Labels = nil
		return builder
	}
	builder.fault.Labels = make(map[string]any, len(labels))
	maps.Copy(builder.fault.Labels, labels)
	return builder
}
