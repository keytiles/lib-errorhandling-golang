# lib-errorhandling-golang

Brings lean but data rich errors into the game. By using this unified error concept you can cut lots of (typically unnecessary and not much useful) complexity
especially in error-mapping often happens between different layers of the application as error is bubbling upwards.

Every error like this is

- Typed, one of `RuntimeFault`, `IllegalStateFault`, `NotImplementedFault`, `ValidationFault`, `ConstraintViolationFault`, `ResourceNotFoundFault`, `AuthenticationFault` or
  `AuthorizationFault` - this kind of the error sets up the main error context (can be mapped nicely into HTTP status code etc e.g.)

- Can carry set of string based error codes - (see predefined constants `*_ERROR_*` in the module) which can be easily extended with custom ones for machine readability

- Can be classified as `IsRetryable()` - yes/no

- Can be decorated with **labels - key-value** pairs

- Has a developers facing message (default) but also can have different audience facing messages optionally.
  all messages can contain variables python-style like “My message with {myVar} variable” which then are resolved from labels.

- Supports collection (lean way!) of source of the error + callStack

- Last but not least, each fault can be classified as `IsPublic()` - yes/no.
  Public faults are OK to leave context boundary and even shown to users as is as this means the error is safe, not leaking internal details for sure and messages are clear.

- Since these errors are data rich **builder pattern** is supported to assemble them convenient way - see `NewFaultBuilder()` and `NewPublicFaultBuilder()`!

A more detailed blog post about the concept most likely will come and when it happens we add the link here.

But until that, start reading here: [Fault interface](pkg/kt_errors/fault.go#fault)
