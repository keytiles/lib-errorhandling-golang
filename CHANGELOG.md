# Versioning policy

We are following [Semantic versioning](https://semver.org/) in this library

We will mark these with Git Tags

# Changes in releases

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
