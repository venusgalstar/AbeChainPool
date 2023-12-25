package utils

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/abesuite/abe-miningpool-server/chaincfg"
)

//	PQRingCT 1.0

//	PQRINGCT1.0 adapts instance-address mechanism.
//	In particular, for a TXO, the coin-address is directly "extracted" from a given instance-address,
//	and it has the following two features:
//	1) the coin-address is a part of the instance-address, and
//	2) the extracting algorithm is deterministic, and the coin-address and instance-address is one-to-one map.

type InstanceAddress struct {
	netID         byte
	cryptoAddress []byte
	cryptoScheme  CryptoScheme
}

func (instAddr *InstanceAddress) SerializeSize() uint32 {
	return uint32(1 + len(instAddr.cryptoAddress))
}

func (instAddr *InstanceAddress) Serialize() []byte {
	b := make([]byte, instAddr.SerializeSize())

	b[0] = instAddr.netID
	copy(b[1:], instAddr.cryptoAddress[:])

	return b
}

func (instAddr *InstanceAddress) Deserialize(serializedInstAddr []byte) error {
	if len(serializedInstAddr) <= 5 {
		return errors.New("byte length of serializedInstAddr does not match the design")
	}

	netId := serializedInstAddr[0]

	//	the bytes [1]~[4] must match the generation of cryptoAddress, namely in pqringctCryptoAddressGen()
	cryptoScheme := serializedInstAddr[1]
	cryptoScheme |= serializedInstAddr[2]
	cryptoScheme |= serializedInstAddr[3]
	cryptoScheme |= serializedInstAddr[4]

	if CryptoScheme(cryptoScheme) != CryptoSchemePQRingCT {
		return errors.New("A non-PQRingCT1.0 abelAddress is deserialized as an instanceAddress")
	}

	//if uint32(len(serializedInstAddr)) != abecrypto.GetCryptoAddressSerializeSize(instAddr.cryptoScheme)+1 {
	//	return errors.New("the length of serializedInstAddr does not match the design")
	//}

	instAddr.netID = netId
	instAddr.cryptoScheme = CryptoScheme(cryptoScheme)

	instAddr.cryptoAddress = make([]byte, len(serializedInstAddr)-1)
	copy(instAddr.cryptoAddress, serializedInstAddr[1:])

	return nil
}

func (instAddr *InstanceAddress) Encode() string {
	serialized := instAddr.Serialize()
	checkSum := DoubleHashH(serialized)

	encodeAddrStr := hex.EncodeToString(serialized)
	encodeAddrStr = encodeAddrStr + hex.EncodeToString(checkSum[:])
	return encodeAddrStr
}

func (instAddr *InstanceAddress) Decode(addrStr string) error {
	addrBytes, err := hex.DecodeString(addrStr)
	if err != nil {
		return err
	}
	if len(addrBytes) <= 5+HashSize {
		errStr := fmt.Sprintf("abel-address has a wrong length")
		return errors.New(errStr)
	}

	serializedInstantAddr := addrBytes[:len(addrBytes)-HashSize]
	checkSum := addrBytes[len(addrBytes)-HashSize:]
	checkSumComputed := DoubleHashH(serializedInstantAddr)
	if bytes.Compare(checkSum, checkSumComputed[:]) != 0 {
		errStr := fmt.Sprintf("abel-address has a wrong check sum")
		return errors.New(errStr)
	}

	err = instAddr.Deserialize(serializedInstantAddr)
	if err != nil {
		return err
	}

	return nil
}

func (instAddr *InstanceAddress) String() string {
	serialized := instAddr.Serialize()
	return hex.EncodeToString(serialized)
}

func (instAddr *InstanceAddress) CryptoAddress() []byte {
	return instAddr.cryptoAddress
}

func (instAddr *InstanceAddress) IsForNet(netParam *chaincfg.Params) bool {
	return instAddr.netID == netParam.PQRingCTID
}
