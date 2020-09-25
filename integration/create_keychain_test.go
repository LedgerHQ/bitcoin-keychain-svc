// +build integration

package integration

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"

	pb "github.com/ledgerhq/bitcoin-keychain-svc/pb/keychain"
)

func TestKeychainRegistration(t *testing.T) {
	ctx := context.Background()
	client, conn := keychainSvcClient(ctx)
	defer conn.Close()

	tests := []struct {
		name            string
		fixture         Fixture
		externalAddress *pb.GetFreshAddressesResponse
		internalAddress *pb.GetFreshAddressesResponse
		wantErr         error
	}{
		{
			name:    "bitcoin mainnet p2pkh",
			fixture: BitcoinMainnetP2PKH,
			externalAddress: &pb.GetFreshAddressesResponse{
				Addresses: []string{"151krzHgfkNoH3XHBzEVi6tSn4db7pVjmR"},
			},
			internalAddress: &pb.GetFreshAddressesResponse{
				Addresses: []string{"13hSrTAvfRzyEcjRcGS5gLEcNVNDhPvvUv"},
			},
		},
		{
			name:    "bitcoin testnet3 p2pkh",
			fixture: BitcoinTestnet3P2PKH,
			externalAddress: &pb.GetFreshAddressesResponse{
				Addresses: []string{"mkpZhYtJu2r87Js3pDiWJDmPte2NRZ8bJV"},
			},
			internalAddress: &pb.GetFreshAddressesResponse{
				Addresses: []string{"mi8nhzZgGZQthq6DQHbru9crMDerUdTKva"},
			},
		},
		{
			name:    "bitcoin testnet3 p2sh-p2wpkh",
			fixture: BitcoinTestnet3P2SHP2WPKH,
			externalAddress: &pb.GetFreshAddressesResponse{
				Addresses: []string{"2MvuUMAG1NFQmmM69Writ6zTsYCnQHFG9BF"},
			},
			internalAddress: &pb.GetFreshAddressesResponse{
				Addresses: []string{"2MsMvWTbPMg4eiSudDa5i7y8XNC8fLCok3c"},
			},
		},
		{
			name:    "bitcoin testnet3 p2wpkh",
			fixture: BitcoinMainnetP2WPKH,
			externalAddress: &pb.GetFreshAddressesResponse{
				Addresses: []string{"bc1qh4kl0a0a3d7su8udc2rn62f8w939prqpl34z86"},
			},
			internalAddress: &pb.GetFreshAddressesResponse{
				Addresses: []string{"bc1qry3crfssh8w6guajms7upclgqsfac4fs4g7nwj"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := client.CreateKeychain(ctx, &pb.CreateKeychainRequest{
				ExtendedPublicKey: tt.fixture.ExtendedPublicKey,
				LookaheadSize:     20,
				Network:           tt.fixture.Network,
				Scheme:            tt.fixture.Scheme,
			})
			if err != nil {
				t.Fatalf("failed to create keychain - error = %v", err)
			}

			wantKeychainInfo := &pb.KeychainInfo{
				KeychainId:              info.KeychainId,
				InternalDescriptor:      tt.fixture.InternalDescriptor,
				ExternalDescriptor:      tt.fixture.ExternalDescriptor,
				ExtendedPublicKey:       tt.fixture.ExtendedPublicKey,
				Slip32ExtendedPublicKey: tt.fixture.ExtendedPublicKey,
				LookaheadSize:           20,
				Scheme:                  tt.fixture.Scheme,
				Network:                 tt.fixture.Network,
			}

			if !proto.Equal(info, wantKeychainInfo) {
				t.Fatalf("CreateKeychain() info = '%v', want = '%v'",
					info, wantKeychainInfo)
			}

			gotExtAddr, err := client.GetFreshAddresses(
				ctx, &pb.GetFreshAddressesRequest{
					KeychainId: info.KeychainId,
					Change:     pb.Change_CHANGE_EXTERNAL,
					BatchSize:  1,
				})
			if err != nil {
				t.Fatalf("failed to get fresh external addr - error = %v", err)
			}

			if !proto.Equal(gotExtAddr, tt.externalAddress) {
				t.Fatalf("GetFreshAddresses() info = '%v', want = '%v'",
					gotExtAddr.Addresses, tt.externalAddress.Addresses)
			}

			gotIntAddr, err := client.GetFreshAddresses(
				ctx, &pb.GetFreshAddressesRequest{
					KeychainId: info.KeychainId,
					Change:     pb.Change_CHANGE_INTERNAL,
					BatchSize:  1,
				})
			if err != nil {
				t.Fatalf("failed to get fresh internal addr - error = %v", err)
			}

			if !proto.Equal(gotIntAddr, tt.internalAddress) {
				t.Fatalf("GetFreshAddresses() info = '%v', want = '%v'",
					gotIntAddr.Addresses, tt.internalAddress.Addresses)
			}
		})
	}
}
