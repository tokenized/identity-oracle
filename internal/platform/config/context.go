package config

import "context"

// ctxKey represents the type of value for the context key.
type ctxKey int

// TestValuesKey is the key for storing/retreiving test values from a context.
const testValuesKey ctxKey = 1

// TestValues represent state for each request.
type TestValues struct {
	RejectQuantity uint64
}

// ContextWithTestValues returns a context with the test values embedded.
func ContextWithTestValues(ctx context.Context, values TestValues) context.Context {
	return context.WithValue(ctx, testValuesKey, values)
}

// ContextTestValues returns the test values associated with the context.
func ContextTestValues(ctx context.Context) TestValues {
	values := ctx.Value(testValuesKey)
	if values == nil {
		return TestValues{RejectQuantity: 0}
	}

	testValues, ok := values.(TestValues)
	if !ok {
		return TestValues{RejectQuantity: 0}
	}
	return testValues
}
