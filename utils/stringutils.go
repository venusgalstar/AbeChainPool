package utils

import (
	"encoding/hex"
	"errors"
	"math/rand"
	"runtime"
	"strconv"
	"time"
	"unicode"

	"github.com/abesuite/abe-miningpool-server/chaincfg"
)

func IsBlank(str string) bool {
	if str == "" {
		return true
	}

	for _, c := range str {
		if !unicode.IsSpace(c) {
			return false
		}
	}
	return true
}

const hashLetters = "abcdef0123456789"

func RandStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = hashLetters[rand.Intn(len(hashLetters))]
	}
	return string(b)
}

const extraNonceLetters = "abcdef0123456789"
const hexStringLetters = "abcdef0123456789"

func RandExtraNonce1(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = extraNonceLetters[rand.Intn(len(extraNonceLetters))]
	}
	return string(b)
}

func RandHexString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = hexStringLetters[rand.Intn(len(hexStringLetters))]
	}
	return string(b)
}

const jobIDLetters = "abcdef0123456789"
const jobIDLen = 8

func GenerateJobID() string {
	b := make([]byte, jobIDLen)
	for i := range b {
		b[i] = extraNonceLetters[rand.Intn(len(jobIDLetters))]
	}
	return string(b)
}

func GetNodeDesc() string {
	systemName := runtime.GOOS
	systemArch := runtime.GOARCH
	goVersion := runtime.Version()
	return "Abec-v" + chaincfg.AbecBackendVersion + "/" + "Pool-v" + chaincfg.PoolBackendVersion + "/" + systemName + "-" + systemArch + "/" + goVersion
}

// CheckAddressValidity checks whether the given address meets the requirement,
// including netID and verification hash
func CheckAddressValidity(addr string, chainParams *chaincfg.Params) error {
	address, err := hex.DecodeString(addr)
	if err != nil {
		return err
	}

	if len(address) < 33 {
		return errors.New("the length of address is incorrect")
	}

	// Check netID
	netID := address[0]
	if netID != chainParams.PQRingCTID {
		return errors.New("address verification fails: the netID of address does not match the active net")
	}

	// Check verification hash
	verifyBytes := address[:len(address)-32]
	dstHash0 := address[len(address)-32:]
	dstHash, err := NewHash(dstHash0)
	if err != nil {
		return err
	}
	realHash := DoubleHashH(verifyBytes)
	if !dstHash.IsEqual(&realHash) {
		return errors.New("address verification fails: verification hash does not match")
	}
	return nil
}

func CheckUsernameAndAddress(user string, addr string) error {
	address, err := hex.DecodeString(addr)
	if err != nil {
		return err
	}

	account := DoubleHashH(address)
	needUser := account.String()
	if needUser != user {
		return errors.New("username and address not match")
	}
	return nil
}

func CalculateUsernameFromAddress(address string) (string, error) {
	addr, err := hex.DecodeString(address)
	if err != nil {
		return "", err
	}
	account := DoubleHashH(addr)
	username := account.String()
	return username, nil
}

var binToHex = map[string]string{
	"0000": "0",
	"0001": "1",
	"0010": "2",
	"0011": "3",
	"0100": "4",
	"0101": "5",
	"0110": "6",
	"0111": "7",
	"1000": "8",
	"1001": "9",
	"1010": "a",
	"1011": "b",
	"1100": "c",
	"1101": "d",
	"1110": "e",
	"1111": "f",
}

var binaryStringLetters = "01"

func GenerateBinary(n int) string {
	b := make([]byte, n)
	l := len(binaryStringLetters)
	for i := range b {
		b[i] = binaryStringLetters[rand.Intn(l)]
	}
	return string(b)
}

func GenerateExtraNonce1(n int) (string, uint64) {
	binaryStr := GenerateBinary(n)
	binaryStrUint64, _ := strconv.ParseUint(binaryStr, 2, 64)
	for len(binaryStr)%4 != 0 {
		binaryStr = binaryStr + "0"
	}
	res := ""
	for i := 0; i < len(binaryStr); i += 4 {
		curr, _ := binToHex[binaryStr[i:i+4]]
		res += curr
	}
	return res, binaryStrUint64
}

func init() {
	rand.Seed(time.Now().Unix())
}
