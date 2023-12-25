package utils

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/abesuite/abe-miningpool-server/constdef"
)

// Number generate given number of int
func Number(size int) []byte {
	if size <= 0 || size > 10 {
		size = 10
	}
	warehouse := []int{48, 57}
	result := make([]byte, size)
	for i := 0; i < size; i++ {
		result[i] = uint8(warehouse[0] + rand.Intn(9))
	}
	return result
}

// Lower generate given number of lower alphabet
func Lower(size int) []byte {
	if size <= 0 || size > 26 {
		size = 26
	}
	warehouse := []int{97, 122}
	result := make([]byte, size)
	for i := 0; i < size; i++ {
		result[i] = uint8(warehouse[0] + rand.Intn(26))
	}
	return result
}

// Upper generate given number of upper alphabet
func Upper(size int) []byte {
	if size <= 0 || size > 26 {
		size = 26
	}
	warehouse := []int{65, 90}
	result := make([]byte, size)
	for i := 0; i < size; i++ {
		result[i] = uint8(warehouse[0] + rand.Intn(26))
	}
	return result
}

// GenerateRandomSalt generate a random string that satisfies the requirements
func GenerateRandomSalt(size int, number int, lower int, upper int) (string, error) {
	length := number + lower + upper
	if size > length || length <= 0 {
		return "", errors.New("invalid salt length")
	}

	var result string
	if lower > 0 {
		lowers := string(Lower(lower))
		result += lowers
	}
	if number > 0 {
		numbers := string(Number(number))
		result += numbers
	}
	if upper > 0 {
		uppers := string(Upper(upper))
		result += uppers
	}

	return result, nil
}

// CheckPasswordValidity checks whether the given password is valid:
// 1. the length of password should between 6-40.
// 2. only letters, numbers and ~!@#$%^&*?_- are allowed
func CheckPasswordValidity(password string) bool {
	passwordLen := len(password)
	if passwordLen < constdef.MinPasswordLength || passwordLen > constdef.MaxPasswordLength {
		return false
	}
	for _, ch := range password {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '~' || ch == '!' || ch == '@' ||
			ch == '#' || ch == '$' || ch == '%' || ch == '^' || ch == '&' || ch == '*' || ch == '?' ||
			ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}

// CheckUsernameValidity checks whether the given username is valid:
// 1. the length of username should between 6-100.
// 2. only letters, numbers and _, @ and . are allowed
func CheckUsernameValidity(username string) bool {
	usernameLen := len(username)
	if usernameLen < constdef.MinUsernameLength || usernameLen > constdef.MaxUsernameLength {
		return false
	}
	for _, ch := range username {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '@' || ch == '_' || ch == '.') {
			return false
		}
	}
	return true
}

// GetInitialShareDifficulty return the initial difficulty using user agent
// The share target is calculated as below
// target = baseTarget / difficulty
func GetInitialShareDifficulty(userAgent string) int64 {
	// todo: Add different difficulty for different device
	if strings.Contains(userAgent, "CPU") {
		return 2
	} else if strings.Contains(userAgent, "GPU") {
		return 1024
	}
	return 1024
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
