package wire

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	"github.com/abesuite/abe-miningpool-server/utils"
)

const (
	blockNumPerRing    = 3
	currMaxRingSize    = 7
	currTxInputMaxNum  = 5
	currTxOutputMaxNum = 5
	MaxBlockPayloadAbe = 800000000
)

// OutPointAbe defines a ABE data type that is used to track previous transaction outputs.
// Note that the type of index depends on the TxOutPutMaxNum
type OutPointAbe struct {
	TxHash utils.Hash
	Index  uint8 //	due to the large size of post-quantum crypto primitives, ABE will limit the number of outputs of each transaction
}

// String returns the OutPoint in the human-readable form "hash:index".
func (op OutPointAbe) String() string {
	// Allocate enough for hash string, colon, and 10 digits.  Although
	// at the time of writing, the number of digits can be no greater than
	// the length of the decimal representation of maxTxOutPerMessage, the
	// maximum message payload may increase in the future and this
	// optimization may go unnoticed, so allocate space for 10 decimal
	// digits, which will fit any uint32.
	buf := make([]byte, 2*utils.HashSize+1, 2*utils.HashSize+1+10)
	copy(buf, op.TxHash.String())
	buf[2*utils.HashSize] = ':'
	buf = strconv.AppendUint(buf, uint64(op.Index), 10)
	return string(buf)
}

type OutPointRing struct {
	Version    uint32        //	All TXOs in a ring has the same version, and this version is set to be the ring Version.
	BlockHashs []*utils.Hash //	the hashs for the blocks from which the ring was generated, at this moment it is 3 successive blocks
	OutPoints  []*OutPointAbe
}

func (outPointRing *OutPointRing) SerializeSize() int {
	// Version 4
	// length of BlockHash
	// BlockHashes Num * utils.HashSize
	// length of Outpoints
	// Outpoints Nun * (utils.HashSize + 1)
	blockNum := len(outPointRing.BlockHashs)
	OutPointNum := len(outPointRing.OutPoints)
	return 4 +
		//VarIntSerializeSize(uint64(blockNum)) +
		1 + //	one byte for blockNum
		blockNum*utils.HashSize +
		//VarIntSerializeSize(uint64(OutPointNum)) +
		1 + //	one byte for OutPointNum
		OutPointNum*(utils.HashSize+1)
}

func (outPointRing *OutPointRing) String() string {
	// Allocate enough for hash string, colon, and 10 digits.  Although
	// at the time of writing, the number of digits can be no greater than
	// the length of the decimal representation of maxTxOutPerMessage, the
	// maximum message payload may increase in the future and this
	// optimization may go unnoticed, so allocate space for 10 decimal
	// digits, which will fit any uint32.
	// TxHash:index; TxHash:index; ...; serialNumber
	//	index is at most 2 decimal digits; at this moment, only 1 decimal digits

	strLen := len(outPointRing.BlockHashs)*(2*utils.HashSize+1+1) + len(outPointRing.OutPoints)*(2*utils.HashSize+1+1+1) + 1
	// Version is not included in the string

	buf := make([]byte, strLen, strLen+11)

	pos := 0

	for _, blockHash := range outPointRing.BlockHashs {
		pos += copy(buf[pos:], blockHash.String())
		buf[pos] = ';'
		pos++
	}

	for _, outPoint := range outPointRing.OutPoints {
		pos += copy(buf[pos:], outPoint.TxHash.String())
		buf[pos] = ','
		pos++

		buf[pos] = outPoint.Index
		pos++

		buf[pos] = ';'
		pos++
	}
	buf[pos] = '.'
	pos++

	return string(buf[0:pos])
}

func (outPointRing *OutPointRing) Hash() utils.Hash {
	buf := bytes.NewBuffer(make([]byte, 0, outPointRing.SerializeSize()))
	_ = WriteOutPointRing(buf, 0, outPointRing.Version, outPointRing)
	return utils.DoubleHashH(buf.Bytes())
}

func WriteOutPointRing(w io.Writer, pver uint32, version uint32, opr *OutPointRing) error {
	err := binarySerializer.PutUint32(w, littleEndian, opr.Version)
	if err != nil {
		return err
	}

	/*	err = WriteVarInt(w, pver, uint64(len(opr.BlockHashs)))
		if err != nil {
			return err
		}*/
	err = binarySerializer.PutUint8(w, uint8(len(opr.BlockHashs)))
	if err != nil {
		return err
	}

	for i := 0; i < len(opr.BlockHashs); i++ {
		_, err := w.Write(opr.BlockHashs[i][:])
		if err != nil {
			return err
		}
	}

	/*	err = WriteVarInt(w, pver, uint64(len(opr.OutPoints)))
		if err != nil {
			return err
		}*/
	err = binarySerializer.PutUint8(w, uint8(len(opr.OutPoints)))
	if err != nil {
		return err
	}
	for i := 0; i < len(opr.OutPoints); i++ {
		err = writeOutPointAbe(w, pver, version, opr.OutPoints[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func ReadOutPointRing(r io.Reader, pver uint32, version uint32, opr *OutPointRing) error {
	err := readElement(r, &opr.Version)
	if err != nil {
		return err
	}

	blockNum, err := binarySerializer.Uint8(r)
	if err != nil {
		return err
	}
	expectedBlockNum := blockNumPerRing
	if blockNum != uint8(expectedBlockNum) {
		str := fmt.Sprintf("the block number %d in ring does not match the version %d", blockNum, opr.Version)
		return messageError("readOutPointRing", str)
	}
	opr.BlockHashs = make([]*utils.Hash, blockNum)
	for i := 0; i < int(blockNum); i++ {
		tmp := utils.Hash{}
		_, err := io.ReadFull(r, tmp[:])
		if err != nil {
			return err
		}
		opr.BlockHashs[i] = &tmp
	}

	//cnt, err = ReadVarInt(r, pver)
	ringSize, err := binarySerializer.Uint8(r)
	if err != nil {
		return err
	}
	maxRingSize := currMaxRingSize
	if err != nil {
		str := fmt.Sprintf("cannot get the ring size with  version %d", opr.Version)
		return messageError("readOutPointRing", str)
	}
	if ringSize > uint8(maxRingSize) {
		str := fmt.Sprintf("the ring size (%d) exceeds the allowed max ring size %d with version %d", ringSize, maxRingSize, opr.Version)
		return messageError("readOutPointRing", str)
	}
	opr.OutPoints = make([]*OutPointAbe, ringSize)
	for i := 0; i < int(ringSize); i++ {
		opr.OutPoints[i] = &OutPointAbe{}
		err = readOutPointAbe(r, pver, version, opr.OutPoints[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func writeOutPointAbe(w io.Writer, pver uint32, version uint32, op *OutPointAbe) error {
	_, err := w.Write(op.TxHash[:])
	if err != nil {
		return err
	}

	return binarySerializer.PutUint8(w, op.Index)
}

func readOutPointAbe(r io.Reader, pver uint32, version uint32, op *OutPointAbe) error {
	_, err := io.ReadFull(r, op.TxHash[:])
	if err != nil {
		return err
	}

	op.Index, err = binarySerializer.Uint8(r)

	return err
}

// TxOutAbe /* As TxOut may be fetched without the corresponding Tx, a version field is used.
type TxOutAbe struct {
	//	Version 		int16	//	the version could be used in ABE protocol update
	// ValueScript   []byte
	//	ValueScript   int64
	//	AddressScript []byte

	Version   uint32
	TxoScript []byte
}

// SerializeSize returns the number of bytes it would take to serialize the
// the transaction output.
func (txOut *TxOutAbe) SerializeSize() int {
	// Value 8 bytes + serialized varint size for the length of AddressScript + AddressScript bytes.
	//return VarIntSerializeSize(uint64(len(txOut.ValueScript))) + len(txOut.ValueScript) + VarIntSerializeSize(uint64(len(txOut.AddressScript))) + len(txOut.AddressScript)
	//return 8 + VarIntSerializeSize(uint64(len(txOut.AddressScript))) + len(txOut.AddressScript)
	return 4 + VarIntSerializeSize(uint64(len(txOut.TxoScript))) + len(txOut.TxoScript)
}

// WriteTxOutAbe encodes ti to the  protocol encoding for a transaction
// input (TxInAbe) to w.
func WriteTxOutAbe(w io.Writer, pver uint32, version uint32, txOut *TxOutAbe) error {
	err := binarySerializer.PutUint32(w, littleEndian, txOut.Version)
	if err != nil {
		return err
	}

	err = WriteVarBytes(w, pver, txOut.TxoScript)
	if err != nil {
		return err
	}

	return nil
}

// ReadTxOutAbe reads the next sequence of bytes from r as a transaction input
// (TxInAbe).
func ReadTxOutAbe(r io.Reader, pver uint32, version uint32, txOut *TxOutAbe) error {
	err := readElement(r, &txOut.Version)
	if err != nil {
		return err
	}

	//	For performance, here the maxallowedlen uses constant, rather than calling funciton.
	txoScript, err := ReadVarBytes(r, pver, 1048576, "TxoScript")
	if err != nil {
		return err
	}
	txOut.TxoScript = txoScript

	return nil
}

// SerialNumber appears only when some ring member is consumed in TxIn,
// i.e. logically, SerialNumber accompanies with TxIn.
type TxInAbe struct {
	SerialNumber []byte
	//	identify the consumed OutPoint
	PreviousOutPointRing OutPointRing
}

// String returns the OutPoint in the human-readable form "hash:index".
func (txIn *TxInAbe) String() string {
	// Allocate enough for hash string, colon, and 10 digits.  Although
	// at the time of writing, the number of digits can be no greater than
	// the length of the decimal representation of maxTxOutPerMessage, the
	// maximum message payload may increase in the future and this
	// optimization may go unnoticed, so allocate space for 10 decimal
	// digits, which will fit any uint32.
	// TxHash:index; TxHash:index; ...; serialNumber
	//	index is at most 2 decimal digits; at this moment, only 1 decimal digits

	snLen := 64
	strLen := 2*snLen + 1 +
		len(txIn.PreviousOutPointRing.BlockHashs)*(2*utils.HashSize+1+1) + len(txIn.PreviousOutPointRing.OutPoints)*(2*utils.HashSize+1+1+1) + 1

	buf := make([]byte, strLen)

	pos := 0

	pos += copy(buf[pos:], hex.EncodeToString(txIn.SerialNumber[:]))
	buf[pos] = '.'
	pos++
	copy(buf[pos:], txIn.PreviousOutPointRing.String())

	return string(buf[0:pos])
}

func (txIn *TxInAbe) SerializeSize() int {
	// utils.HashSize for SerialNumber
	//snLen := abepqringctparam.GetTxoSerialNumberLen(txIn.PreviousOutPointRing.Version)
	snLen := len(txIn.SerialNumber)
	return VarIntSerializeSize(uint64(snLen)) + snLen + txIn.PreviousOutPointRing.SerializeSize()
}

// writeTxIn encodes txIn to the  protocol encoding for a transaction
// input (TxIn) to w.
func writeTxInAbe(w io.Writer, pver uint32, version uint32, txIn *TxInAbe) error {
	err := WriteVarBytes(w, pver, txIn.SerialNumber)
	if err != nil {
		return err
	}

	return WriteOutPointRing(w, pver, version, &txIn.PreviousOutPointRing)
}

// readTxIn reads the next sequence of bytes from r as a transaction input
// (TxIn).
func readTxInAbe(r io.Reader, pver uint32, version uint32, txIn *TxInAbe) error {
	var err error

	// For better performance, here we use constant to specify the maxallowedsize, rather than calling a function.
	txIn.SerialNumber, err = ReadVarBytes(r, pver, 64, "SerialNumber")
	if err != nil {
		return err
	}
	return ReadOutPointRing(r, pver, version, &txIn.PreviousOutPointRing)
}

// At this moment, TxIn is just a ring member (say, identified by (outpointRing, serialNumber)), so that we can use txIn.Serialize
// If TxIn includes more information, this needs modification.
func (txIn *TxInAbe) RingMemberHash() utils.Hash {
	buf := bytes.NewBuffer(make([]byte, 0, txIn.PreviousOutPointRing.SerializeSize()))
	_ = WriteOutPointRing(buf, 0, txIn.PreviousOutPointRing.Version, &txIn.PreviousOutPointRing)
	return utils.DoubleHashH(buf.Bytes())
}

type MsgTxAbe struct {
	Version uint32
	TxIns   []*TxInAbe

	TxOuts []*TxOutAbe

	TxFee uint64

	//	At most 1024 bytes
	TxMemo []byte

	TxWitness []byte
}

// AddTxIn adds a transaction input to the message.
func (msg *MsgTxAbe) AddTxIn(txIn *TxInAbe) {
	msg.TxIns = append(msg.TxIns, txIn)
}

// AddTxOut adds a transaction output to the message.
func (msg *MsgTxAbe) AddTxOut(txOut *TxOutAbe) {
	msg.TxOuts = append(msg.TxOuts, txOut)
}

// HasWitness returns false if none of the inputs within the transaction
// contain witness data, true false otherwise.
func (msg *MsgTxAbe) HasWitness() bool {
	if msg.TxWitness == nil || len(msg.TxWitness) == 0 {
		return false
	}
	return true
}

func GetNullSerialNumber() []byte {
	snSize := 64
	nullSn := make([]byte, snSize)
	for i := 0; i < snSize; i++ {
		nullSn[i] = 0
	}
	return nullSn
}

func (msg *MsgTxAbe) IsCoinBase() (bool, error) {
	if len(msg.TxIns) != 1 {
		return false, nil
	}

	// The serialNumber of the consumed coin must be a zero hash.
	// Whatever ths ring members for the TxIns[0]
	// the ring members' (TXHash, index) can be used as coin-nonce
	txIn := msg.TxIns[0]
	nullSn := GetNullSerialNumber()
	if bytes.Compare(txIn.SerialNumber, nullSn) != 0 {
		return false, nil
	}

	return true, nil
}

// TxHash generates the Hash for the transaction without witness.
func (msg *MsgTxAbe) TxHash() utils.Hash {
	// Encode the transaction and calculate double sha256 on the result.
	// Ignore the error returns since the only way the encode could fail
	// is being out of memory or due to nil pointers, both of which would
	// cause a run-time panic.
	buf := bytes.NewBuffer(make([]byte, 0, msg.SerializeSize()))
	_ = msg.Serialize(buf)
	return utils.DoubleHashH(buf.Bytes())
}

// TxHashFull generates the Hash for the transaction with witness.
func (msg *MsgTxAbe) TxHashFull() utils.Hash {
	buf := bytes.NewBuffer(make([]byte, 0, msg.SerializeSizeFull()))
	_ = msg.SerializeFull(buf)
	return utils.DoubleHashH(buf.Bytes())
}

// BtcDecode decodes r using the  protocol encoding into the receiver.
// This is part of the Message interface implementation.
// See Deserialize for decoding transactions stored to disk, such as in a
// database, as opposed to decoding transactions from the wire.
func (msg *MsgTxAbe) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	version, err := binarySerializer.Uint32(r, littleEndian)
	if err != nil {
		return err
	}
	msg.Version = version

	//	TxIns
	txInNum, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	txInputMaxNum := currTxInputMaxNum
	if txInNum > uint64(txInputMaxNum) {
		str := fmt.Sprintf("The numner of inputs exceeds the allowd max number [txInNum %d, max %d]", txInNum,
			txInputMaxNum)
		return messageError("MsgTx.BtcDecode", str)
	}
	msg.TxIns = make([]*TxInAbe, txInNum)
	for i := uint64(0); i < txInNum; i++ {
		txIn := TxInAbe{}
		err = readTxInAbe(r, pver, msg.Version, &txIn)
		if err != nil {
			return err
		}
		msg.TxIns[i] = &txIn
	}

	// TxOuts
	txoNum, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	txOutputMaxNum := currTxOutputMaxNum
	if txoNum > uint64(txOutputMaxNum) {
		str := fmt.Sprintf("The numner of inputs exceeds the allowd max number [txInNum %d, max %d]", txoNum,
			txOutputMaxNum)
		return messageError("MsgTx.BtcDecode", str)
	}
	msg.TxOuts = make([]*TxOutAbe, txoNum)
	for i := uint64(0); i < txoNum; i++ {
		txOut := TxOutAbe{}
		err = ReadTxOutAbe(r, pver, msg.Version, &txOut)
		if err != nil {
			return err
		}
		msg.TxOuts[i] = &txOut
	}

	//	TxFee
	txFee, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	msg.TxFee = txFee

	//	TxMemo
	//	For better performance, we use constant to specify the maxallowed size, rather than calling a function.
	// txMemo, err := ReadVarBytes(r, pver, uint32(abepqringctparam.GetTxMemoMaxLen(msg.Version)), "TxMemo")
	txMemo, err := ReadVarBytes(r, pver, 1024, "TxMemo")
	if err != nil {
		return err
	}
	msg.TxMemo = txMemo

	if enc == WitnessEncoding {
		//	TxWitness
		//	For better performance, we use constant to specify the maxallowed size, rather than calling a function.
		// txWitness, err := ReadVarBytes(r, pver, uint32(abepqringctparam.GetTxWitnessMaxLen(msg.Version)), "TxWitness")
		txWitness, err := ReadVarBytes(r, pver, 16777216, "TxWitness")
		if err != nil {
			msg.TxWitness = nil
		}
		msg.TxWitness = txWitness
	}

	return nil
}

// BtcEncode encodes the receiver to w using the  protocol encoding.
// This is part of the Message interface implementation.
// See Serialize for encoding transactions to be stored to disk, such as in a
// database, as opposed to encoding transactions for the wire.
func (msg *MsgTxAbe) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	//	Version
	err := binarySerializer.PutUint32(w, littleEndian, msg.Version)
	if err != nil {
		return err
	}

	//	TxIns
	err = WriteVarInt(w, 0, uint64(len(msg.TxIns)))
	if err != nil {
		return err
	}
	for _, txIn := range msg.TxIns {
		err = writeTxInAbe(w, pver, msg.Version, txIn)
		if err != nil {
			return err
		}
	}

	// TxOuts
	err = WriteVarInt(w, 0, uint64(len(msg.TxOuts)))
	for _, txOut := range msg.TxOuts {
		err = WriteTxOutAbe(w, pver, msg.Version, txOut)
		if err != nil {
			return err
		}
	}

	//	TxFee
	err = WriteVarInt(w, 0, msg.TxFee)
	if err != nil {
		return err
	}

	//	TxMemo
	err = WriteVarBytes(w, 0, msg.TxMemo)
	if err != nil {
		return err
	}

	if enc == WitnessEncoding && msg.HasWitness() {
		err = WriteVarBytes(w, 0, msg.TxWitness)
		if err != nil {
			return err
		}
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgTxAbe) Command() string {
	return CmdTx
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgTxAbe) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayloadAbe
}

// Note that, in different system, int may be int16 or int32
func (msg *MsgTxAbe) SerializeSize() int {
	//	Version 4 bytes
	n := 4

	//	Inputs
	//	serialized varint size for input
	n = n + VarIntSerializeSize(uint64(len(msg.TxIns)))
	for _, txIn := range msg.TxIns {
		// serialized varint size for the ring size, and (utils.HashSize + 1) for each OutPoint
		n = n + txIn.SerializeSize()
	}

	// 	serialized varint size for output number
	n = n + VarIntSerializeSize(uint64(len(msg.TxOuts)))
	for _, txOut := range msg.TxOuts {
		n = n + txOut.SerializeSize()
	}

	//	TxFee
	n = n + VarIntSerializeSize(msg.TxFee)
	//	use 8 bytes store the transaction fee
	//n = n + 8

	//	TxMemo
	n = n + VarIntSerializeSize(uint64(len(msg.TxMemo))) + len(msg.TxMemo)

	return n
}

func (msg *MsgTxAbe) SerializeSizeFull() int {
	n := msg.SerializeSize()

	n = n + VarIntSerializeSize(uint64(len(msg.TxWitness))) + len(msg.TxWitness)

	return n
}

// for computing TxId by hash, and for being serialized in block
func (msg *MsgTxAbe) Serialize(w io.Writer) error {
	return msg.BtcEncode(w, 0, BaseEncoding)
}

func (msg *MsgTxAbe) SerializeFull(w io.Writer) error {
	return msg.BtcEncode(w, 0, WitnessEncoding)
}

// Deserialize decodes a transaction from r into the receiver using a format
// that is suitable for long-term storage such as a database while respecting
// the Version field in the transaction.  This function differs from BtcDecode
// in that BtcDecode decodes from the  wire protocol as it was sent
// across the network.  The wire encoding can technically differ depending on
// the protocol version and doesn't even really need to match the format of a
// stored transaction at all.  As of the time this comment was written, the
// encoded transaction is the same in both instances, but there is a distinct
// difference and separating the two allows the API to be flexible enough to
// deal with changes.
func (msg *MsgTxAbe) Deserialize(r io.Reader) error {
	return msg.BtcDecode(r, 0, WitnessEncoding)
}
func (msg *MsgTxAbe) DeserializeNoWitness(r io.Reader) error {
	return msg.BtcDecode(r, 0, BaseEncoding)
}

func (msg *MsgTxAbe) DeserializeFull(r io.Reader) error {
	return msg.BtcDecode(r, 0, WitnessEncoding)
}
