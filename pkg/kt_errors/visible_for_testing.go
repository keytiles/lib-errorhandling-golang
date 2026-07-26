package kt_errors

// VisibleForTesting_NilFault returns a typed-nil Fault (*defaultFault(nil) held in the Fault interface).
// Used by tests to verify nil-safe Error()/String() behavior that a plain nil interface value cannot cover.
func VisibleForTesting_NilFault() Fault {
	var fault *defaultFault
	return fault
}
