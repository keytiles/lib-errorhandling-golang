# Versioning policy

We are following [Semantic versioning](https://semver.org/) in this library

We will mark these with Git Tags

# Changes in releases

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
