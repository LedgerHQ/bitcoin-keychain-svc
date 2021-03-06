package bitcoin

import "github.com/pkg/errors"

var (
	// ErrUnrecognizedAddressEncoding indicates that an unrecognized encoding
	// for addresses was encountered in a bitcoin-lib-grpc payload.
	ErrUnrecognizedAddressEncoding = errors.New("unrecognized address encoding")

	// ErrUnrecognizedNetwork indicates that an unknown network was encountered
	// in a bitcoin-lib-grpc payload.
	ErrUnrecognizedNetwork = errors.New("unrecognized network")
)
