# Versioning policy

We are following [Semantic versioning](https://semver.org/) in this library

We will mark these with Git Tags

## release 2.0.1

Fixes:

- Making `kt_errors.conversionOption` iface public so renamed to `kt_errors.ConversionOption` - this is necessary so other libs can create wrappers around the conversion.

## release 2.0.0

Breaking changes:

- Changed the signature of utility function `kt_errors.NewPublicFaultFromAnyError()`. From now it is much more convenient to fine tune a bit how the conversion
  works using the `options` parameter. We introduced a few non-mandatory but super useful options - see `kt_errors.OptionXXX()` methods!
- Because of the above we do not need the `kt_errors.NO_LOG_LABELS` constant anymore - so it was removed.

## release 1.3.8

Fixes:

- Forgot to add `kt_errors.AUTHENTICATION_ERRCODE_INVALID` error code - now added.

## release 1.3.7

Fixes:

- Forgot to add `kt_errors.AUTHENTICATION_ERRCODE_EXPIRED` error code - now added.

## release 1.3.6

Fixes:

- Making default `Fault` implementation Nil pointer safe - until this time it would have paniced if one tried to invoke any method on a Nil Fault instance

## release 1.3.5

Skipped - we made a git tag error and not possible to fix in place

## release 1.3.4

Fixes:

- `kt_errors.GetGrpcStatusCodeForFault` and `kt_errors.GetHttpStatusCodeForFault` did not classify correctly (=OK) if passed Fault parameter was nil. Now it is fixed.
  And also more test cases added to verify behavior of these functions.

# Changes in releases

## release 1.3.3

Fixes:

- Forgot to add two important build in IllegalState error code which often happens: `ILLEGALSTATE_ERRCODE_SERIALIZATION_FAILED` and `ILLEGALSTATE_ERRCODE_DESERIALIZATION_FAILED`

## release 1.3.2

Fixes:

- Oops forgot to add `fault.GetLabel()` method. Without this people need to invoke `fault.GetLabels()` and query a full copy - not optimal...
  So now this method is available.

## release 1.3.1

Fixes:

- Just internal restructuring - switching errorCodes, labels and audience message arrays/maps into lazy init mode. They get created only when really used but
  until this they remain Nil.
- New test cases added to check functionality.

## release 1.3.0

Fixes:

- `FaultBuilder` got mutated when built `Fault` was mutated e.g. with adding labels. This happened because instead of passing a map copy into the Fault we
  passed the map by reference. Now this is fixed and several test cases added to test this behavior.
- `fault.ToFullJSON()` serializer was incorrect - it mutated the original Fault if features like `ResolveMessages` was applied. Now it is fixed and test cases
  were added to make sure orig Fault is never mutated accidentally

New features:

- From now on the JSON serializers `fault.ToFullJSON()` or `fault.ToNaturalJSON()` if `ResolveMessages` option applied will remove those {var} entries from
  the labels section (in the JSON) which are used in message / audience message. As typically those vars are added to labels for this resolving possibility
  and if the msgs already resolved you probably don't want them there too anymore (as they became part of the text). But if you do, a new option was
  introduced `LeaveMessageVarsInLabels` which prevents this behavior.

## release 1.2.0

New features:

- Added support for HTTP and gRPC status codes derived from the Fault. See
  - utility functions `kt_errors.GetHttpStatusCodeForFault()` and `kt_errors.GetGrpcStatusCodeForFault()`, or
  - member functions `fault.GetHttpStatusCode()` and `fault.GetGrpcStatusCode()`
- Added simple support (OK not that bad) to marshal the Fault into JSON forms which could come handy for rapid API and error return developments. See
  - utility functions `kt_errors.GetFaultAsNaturalJSON()` and `kt_errors.GetFaultAsFullJSON()`, or
  - member functions `fault.ToNaturalJSON()` and `fault.ToFullJSON()`

## release 1.1.1

Fixes:

- Fixed default logger name to match with library name: "keytiles.errorhandling"
- During `kt_errors.NewPublicFaultFromAnyError()` utility method conversion if you provided `transactionId` param now it is always added as "transactionId"
  label to the converted Fault.

## release 1.1.0

First practical experiences from adoption quickly showed some new things can be useful.

New features:

- From now on `Fault` became even a bit more mutable. As error bubbling upwards the call chain higher level layers might want to extend the `Fault` with more
  context and information (mutate it a bit) without the need of re-creating a brand new instance from scratch (which often leads lots of boilerplate).
  Just focus on what really matters!
  New functions added
  - `AddContextToMessage()` - read its comment!
  - `AddContextToAudienceMessage()`
  - `AddErrorCodes()`
  - `AddLabel()` and `AddLabels()`

## release 1.0.0

Initial usable release
