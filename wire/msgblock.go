package wire

import (
	"io"

	"github.com/abesuite/abe-miningpool-server/utils"
)

// MsgSimplifiedBlock is used to submit simplified block (without transaction content)
// Transactions contains coinbase (the first one)
type MsgSimplifiedBlock struct {
	Header       BlockHeader
	Coinbase     *MsgTxAbe
	Transactions []utils.Hash
}

func (msg *MsgSimplifiedBlock) AbeDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := readBlockHeader(r, pver, &msg.Header)
	if err != nil {
		return err
	}

	coinbaseExist, err := ReadVarInt(r, pver)
	if coinbaseExist == 1 {
		tx := MsgTxAbe{}
		err := tx.BtcDecode(r, pver, enc)
		if err != nil {
			return err
		}
		msg.Coinbase = &tx
	} else {
		msg.Coinbase = nil
	}

	txCount, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

	msg.Transactions = make([]utils.Hash, txCount)
	for i := 0; i < int(txCount); i++ {
		_, err := io.ReadFull(r, msg.Transactions[i][:])
		if err != nil {
			return err
		}
	}

	return nil
}

func (msg *MsgSimplifiedBlock) Deserialize(r io.Reader) error {
	return msg.AbeDecode(r, 0, WitnessEncoding)
}

func (msg *MsgSimplifiedBlock) Serialize(w io.Writer) error {
	return msg.AbeEncode(w, 0, WitnessEncoding)
}

func (msg *MsgSimplifiedBlock) AbeEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := writeBlockHeader(w, pver, &msg.Header)
	if err != nil {
		return err
	}

	if msg.Coinbase == nil {
		err = WriteVarInt(w, pver, 0)
		if err != nil {
			return err
		}
	} else {
		err = WriteVarInt(w, pver, 1)
		if err != nil {
			return err
		}
		err = msg.Coinbase.BtcEncode(w, pver, enc)
		if err != nil {
			return err
		}
	}

	err = WriteVarInt(w, pver, uint64(len(msg.Transactions)))
	if err != nil {
		return err
	}

	for _, txHash := range msg.Transactions {
		_, err = w.Write(txHash[:])
		if err != nil {
			return err
		}
	}

	return nil
}
