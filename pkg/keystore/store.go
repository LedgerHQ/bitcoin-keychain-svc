package keystore

import (
	"fmt"

	"github.com/pkg/errors"
)

// Keystore is an interface that all keychain storage backends must implement.
// Currently, there are two Keystore implementations available:
//   InMemoryKeystore: useful for unit-tests
//   RedisKeystore:    TBA
type Keystore interface {
	Get(descriptor string) (KeychainInfo, error)
	Create(descriptor string, net Network) (KeychainInfo, error)
	GetFreshAddress(descriptor string, change Change) (string, error)
	GetFreshAddresses(descriptor string, change Change, size uint32) ([]string, error)
	MarkPathAsUsed(descriptor string, path DerivationPath) error
	GetAllObservableIndexes(descriptor string, change Change, from uint32, to uint32) ([]uint32, error)
}

// Scheme defines the scheme on which a keychain entry is based.
type Scheme string

const (
	// BIP44 indicates that the keychain scheme is legacy.
	BIP44 Scheme = "BIP44"

	// BIP49 indicates that the keychain scheme is segwit.
	BIP49 Scheme = "BIP49"

	// BIP84 indicates that the keychain scheme is native segwit.
	BIP84 Scheme = "BIP84"
)

// Network defines the network (and therefore the chain parameters)
// that a keychain is associated to.
type Network string

const (
	// Mainnet indicates the main Bitcoin network
	Mainnet Network = "mainnet"

	// Testnet3 indicates the current Bitcoin test network
	Testnet3 Network = "testnet3"

	// Regtest indicates the Bitcoin regression test network
	Regtest Network = "regtest"
)

const lookaheadSize = 20

// KeychainInfo models the global information related to an account registered
// in the keystore.
//
// Rather than using the associated gRPC message struct, it is defined here
// independently to avoid having gRPC dependency in this package.
type KeychainInfo struct {
	Descriptor                    string   `json:"descriptor"`
	XPub                          string   `json:"xpub"`                             // Extended public key serialized with standard HD version bytes
	SLIP32ExtendedPublicKey       string   `json:"slip32_extended_public_key"`       // Extended public key serialized with SLIP-0132 HD version bytes
	ExternalXPub                  string   `json:"external_xpub"`                    // External chain extended public key at HD tree depth 4
	MaxConsecutiveExternalIndex   uint32   `json:"max_consecutive_external_index"`   // Max consecutive index (without any gap) on the external chain
	InternalXPub                  string   `json:"internal_xpub"`                    // Internal chain extended public key at HD tree depth 4
	MaxConsecutiveInternalIndex   uint32   `json:"max_consecutive_internal_index"`   // Max consecutive index (without any gap) on the internal chain
	LookaheadSize                 uint32   `json:"lookahead_size"`                   // Numerical size of the lookahead zone
	Scheme                        Scheme   `json:"scheme"`                           // String identifier for keychain scheme
	Network                       Network  `json:"network"`                          // String denoting the network to use for encoding addresses
	NonConsecutiveExternalIndexes []uint32 `json:"non_consecutive_external_indexes"` // Used external indexes that are creating a gap in the derivation
	NonConsecutiveInternalIndexes []uint32 `json:"non_consecutive_internal_indexes"` // Used internal indexes that are creating a gap in the derivation
}

type derivationToPublicKeyMap map[DerivationPath]struct {
	PublicKey string `json:"public_key"` // Public key at HD tree depth 5
	Used      bool   `json:"used"`       // Whether any txn history at derivation
}

// Schema is a map between account descriptors and account information.
type Schema map[string]*Meta

// Meta is a struct containing account details corresponding to a descriptor,
// such as derivations, addresses, etc.
type Meta struct {
	Main        KeychainInfo              `json:"main"`
	Derivations derivationToPublicKeyMap  `json:"derivations"`
	Addresses   map[string]DerivationPath `json:"addresses"` // derivation path at HD tree depth 5
}

// ChangeXPub returns the XPub of the keychain for the specified Change
// (Internal or External).
func (m Meta) ChangeXPub(change Change) (string, error) {
	switch change {
	case External:
		return m.Main.ExternalXPub, nil
	case Internal:
		return m.Main.InternalXPub, nil
	default:
		return "", errors.Wrapf(ErrUnrecognizedChange, fmt.Sprint(change))
	}
}

// MaxConsecutiveIndex returns the max consecutive index without any gap,
// for the specified Change (Internal or External).
func (m Meta) MaxConsecutiveIndex(change Change) (uint32, error) {
	switch change {
	case External:
		return m.Main.MaxConsecutiveExternalIndex, nil
	case Internal:
		return m.Main.MaxConsecutiveInternalIndex, nil
	default:
		return 0, errors.Wrapf(ErrUnrecognizedChange, fmt.Sprint(change))
	}
}

// SetMaxConsecutiveIndex updates the max consecutive index value for the
// specified Change (Internal or External).
func (m *Meta) SetMaxConsecutiveIndex(change Change, index uint32) error {
	switch change {
	case External:
		m.Main.MaxConsecutiveExternalIndex = index
	case Internal:
		m.Main.MaxConsecutiveInternalIndex = index
	default:
		return errors.Wrapf(ErrUnrecognizedChange, fmt.Sprint(change))
	}

	return nil
}

// NonConsecutiveIndexes returns the non-consecutive indexes introduced due to
// gaps in derived addresses, for the specified Change (Internal or External).
func (m Meta) NonConsecutiveIndexes(change Change) ([]uint32, error) {
	switch change {
	case External:
		return m.Main.NonConsecutiveExternalIndexes, nil
	case Internal:
		return m.Main.NonConsecutiveInternalIndexes, nil
	default:
		return nil, errors.Wrapf(ErrUnrecognizedChange, fmt.Sprint(change))
	}
}

// SetNonConsecutiveIndexes updates the non-consecutive indexes for the
// specified Change (Internal or External).
//
// Any index less than the max consecutive index will be filtered out to handle
// the case when a previously introduced gap is filled.
func (m *Meta) SetNonConsecutiveIndexes(change Change, indexes []uint32) error {
	maxConsecutiveIndex, err := m.MaxConsecutiveIndex(change)
	if err != nil {
		return err
	}

	var result []uint32

	// Filter out all non-consecutive indexes less than the max consecutive
	// index.
	for _, i := range indexes {
		if i >= maxConsecutiveIndex {
			result = append(result, i)
		}
	}

	switch change {
	case External:
		m.Main.NonConsecutiveExternalIndexes = result
	case Internal:
		m.Main.NonConsecutiveInternalIndexes = result
	default:
		return errors.Wrapf(ErrUnrecognizedChange, fmt.Sprint(change))
	}

	return nil
}

func (m Meta) MaxObservableIndex(change Change) (uint32, error) {
	switch change {
	case External:
		n := uint32(len(m.Main.NonConsecutiveExternalIndexes))
		return m.Main.MaxConsecutiveExternalIndex + n + m.Main.LookaheadSize, nil
	case Internal:
		n := uint32(len(m.Main.NonConsecutiveInternalIndexes))
		return m.Main.MaxConsecutiveInternalIndex + n + m.Main.LookaheadSize, nil
	default:
		return 0, errors.Wrapf(ErrUnrecognizedChange, fmt.Sprint(change))
	}
}
