package handlers

// translate looks for certain error types and transforms
// them into web errors. We are losing the trace when this
// error is converted. But we don't log traces for these.
func translate(err error) error {
	return err
}
