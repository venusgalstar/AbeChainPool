package utils

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/abesuite/abe-miningpool-server/chaincfg"
)

//	ABE can support at most 2^32 different CryptoSchemes
//	The different versions of 'same' CryptoSchemes are regarded as different CryptoSchemes.
type CryptoScheme uint32

const (
	CryptoSchemePQRingCT CryptoScheme = iota
)

// AbelAddress is the address facing to the users.
// In particular, its EncodeString form is used by users to receive coins.
// AbelAddress vs. CryptoAddress:
// 1. CryptoAddress is generated and used by the underlying crypto scheme, such as PQRingCT1.0.
// 2. AbelAddress's string form is used by users to receives coins, such as mining-address or payment-address.
// As a result, AbelAddress instance shall contain (netId, cryptoAddress) and its encodeString form shall contain checksum.
type AbelAddress interface {
	SerializeSize() uint32

	// Serialize returns the bytes for (netId, cryptoAddress)
	Serialize() []byte

	// Deserialize parses the bytes for (netId, cryptoAddress) to AbelAddress object.
	Deserialize(serialized []byte) error

	//	Encode() returns hex-codes of (netId, cryptoAddress, checksum) where checksum is the hash of (netId, cryptoAddress)
	Encode() string

	//	Decode() parses the hex-codes of (netId, cryptoAddress, checksum) to AbelAddress object.
	Decode(addrStr string) error

	//	String() returns the hex-codes of (netId, cryptoAddress)
	String() string

	//	CryptoAddress() returns the cryptoAddress
	CryptoAddress() []byte

	// IsForNet returns whether or not the address is associated with the passed Abelian network.
	IsForNet(*chaincfg.Params) bool
}

// DecodeAbelAddress calls the concrete AbelAddress type's Decode() method, based on the cryptoScheme.
// All AbelAddress types shall obey the same rule on the first 5 bytes of its encode, naemly
// (1) [0] specifies the AbelAddressNetId, say 0x00 for mainnet, 0x01 for ...
// (2) [1]~[4] specifies the CryptoScheme of the corresponding CryptoAddress, which are encoded in abecrypto.CryptoAddressKeyGen()
func DecodeAbelAddress(addrStr string) (AbelAddress, error) {
	addrBytes, err := hex.DecodeString(addrStr)
	if err != nil {
		return nil, err
	}
	if len(addrBytes) < 5 {
		errStr := fmt.Sprintf("abel-address has a wrong length")
		return nil, errors.New(errStr)
	}

	//	the bytes [1]~[4] must match the generation of cryptoAddress, namely in pqringctCryptoAddressGen()
	cryptoScheme := addrBytes[1]
	cryptoScheme |= addrBytes[2]
	cryptoScheme |= addrBytes[3]
	cryptoScheme |= addrBytes[4]

	switch CryptoScheme(cryptoScheme) {
	case CryptoSchemePQRingCT:
		instAddr := &InstanceAddress{}
		err := instAddr.Decode(addrStr)
		if err != nil {
			return nil, err
		}

		return instAddr, nil

	default:
		return nil, errors.New("unsupported cryptoScheme of AbelAddress")
	}

}
