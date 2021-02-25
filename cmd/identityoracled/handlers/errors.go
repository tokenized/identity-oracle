package handlers

import (
	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/web"

	"github.com/pkg/errors"
)

// translate looks for certain error types and transforms
// them into web errors. We are losing the trace when this
// error is converted. But we don't log traces for these.
func translate(err error) error {
	switch errors.Cause(err) {
	case oracle.ErrXPubNotFound:
		return errors.Wrap(web.ErrNotFound, err.Error())
	case oracle.ErrUserNotFound:
		return errors.Wrap(web.ErrNotFound, err.Error())
	case oracle.ErrInvalidSignature:
		return errors.Wrap(web.ErrUnauthorized, err.Error())
	}
	return err
}
