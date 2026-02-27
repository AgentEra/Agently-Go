package types

// OutputTuple models Python output tuple semantics used by configure_prompt:
// (type, desc, ...). It is intentionally represented as a slice-like type so
// generic sanitizers serialize nested values in a Python-compatible way.
type OutputTuple []any

func NewOutputTuple(typeValue any, descValue any) OutputTuple {
	return OutputTuple{typeValue, descValue}
}
