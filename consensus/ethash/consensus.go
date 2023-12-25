package ethash

import (
	"bytes"
	"encoding/binary"
	"errors"
	"golang.org/x/crypto/sha3"
	"math/big"
	"runtime"
	"time"

	"github.com/abesuite/abe-miningpool-server/chaincfg"
	"github.com/abesuite/abe-miningpool-server/utils"
	"github.com/abesuite/abe-miningpool-server/wire"
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errInvalidMixDigest = errors.New("invalid mix digest")
	errInvalidPoW       = errors.New("invalid proof-of-work")
)

// epoch() packages the computation of epoch.
// To be efficient, when calling this function inside ethash package, we directly use the computation rather than call this function.
func epoch(blockHeight int32) int {
	return int((blockHeight - chaincfg.ActiveNetParams.BlockHeightEthashPoW) / chaincfg.ActiveNetParams.EthashEpochLength)
}

func CalculateEpoch(blockHeight int32) int {
	if blockHeight-chaincfg.ActiveNetParams.BlockHeightEthashPoW < 0 {
		return -1
	}
	return int((blockHeight - chaincfg.ActiveNetParams.BlockHeightEthashPoW) / chaincfg.ActiveNetParams.EthashEpochLength)
}

func CalculateEpochStartHeight(epoch int) int {
	return int(chaincfg.ActiveNetParams.BlockHeightEthashPoW) + epoch*int(chaincfg.ActiveNetParams.EthashEpochLength)
}

// VerifySeal checks whether a block(header) satisfies the PoW difficulty requirements,
// either using the usual ethash cache for it, or alternatively using a full DAG to make it faster.
//
//	VerifySeal() define the consensus rules.
//	As the target rule may depend on the block chain, Ethash leaves the difficulty computation to block chain module.
//
// func (ethash *Ethash) VerifySeal(header *wire.BlockHeader, target *big.Int, fulldag bool) error {
func (ethash *Ethash) VerifySeal(header *wire.BlockHeader, target *big.Int) error {

	//	ethash.config.VerifyByFullDAG allows to configure a mining machine to use fullDAG to have faster verification.
	fulldag := ethash.config.VerifyByFullDAG

	// If we're running a fake PoW, accept any seal as valid
	if ethash.config.PowMode == ModeFake || ethash.config.PowMode == ModeFullFake {
		time.Sleep(ethash.fakeDelay)
		if ethash.fakeFail == header.Height {
			return errInvalidPoW
		}
		return nil
	}

	//epoch := epoch(header.Height)
	epoch := int((header.Height - chaincfg.ActiveNetParams.BlockHeightEthashPoW) / chaincfg.ActiveNetParams.EthashEpochLength)

	// Recompute the digest and PoW values
	var (
		digest   []byte
		sealHash utils.Hash
	)

	// If fast-but-heavy PoW verification was requested, use an ethash dataset
	if fulldag {
		dataset := ethash.dataset(epoch, true)
		if dataset.generated() {
			digest, sealHash = hashimotoFull(dataset.dataset, header.ContentHash(), header.NonceExt)

			// Datasets are unmapped in a finalizer. Ensure that the dataset stays alive
			// until after the call to hashimotoFull so it's not unmapped while being used.
			runtime.KeepAlive(dataset)
		} else {
			// dataset not yet generated, don't hang, use a cache instead
			fulldag = false
		}
	}
	// If slow-but-light PoW verification was requested (or DAG not yet ready), use an ethash cache
	if !fulldag {
		cache := ethash.cache(epoch)

		size := datasetSize(epoch)
		if ethash.config.PowMode == ModeTest {
			size = 32 * 1024
		}
		digest, sealHash = hashimotoLight(size, cache.cache, header.ContentHash(), header.NonceExt)

		// Caches are unmapped in a finalizer. Ensure that the cache stays alive
		// until after the call to hashimotoLight so it's not unmapped while being used.
		runtime.KeepAlive(cache)
	}

	// Verify the calculated values against the ones provided in the header
	if !bytes.Equal(header.MixDigest[:], digest) {
		return errInvalidMixDigest
	}

	if hashToBig(sealHash).Cmp(target) > 0 {
		return errInvalidPoW
	}

	////	Optimization: directly hashToBig here, rather call the function hashToBig()
	////	The mining part codes, e.g. TrySeal(), should match the codes here.
	//for i := 0; i < utils.HashSize/2; i++ {
	//	sealHash[i], sealHash[utils.HashSize-1-i] = sealHash[utils.HashSize-1-i], sealHash[i]
	//}
	//if new(big.Int).SetBytes(sealHash[:]).Cmp(target) > 0 {
	//	return errInvalidPoW
	//}

	return nil
}

func (ethash *Ethash) CalculateSeal(header *wire.BlockHeader) (utils.Hash, []byte, error) {
	//	ethash.config.VerifyByFullDAG allows to configure a mining machine to use fullDAG to have faster verification.
	fulldag := ethash.config.VerifyByFullDAG

	epoch := int((header.Height - chaincfg.ActiveNetParams.BlockHeightEthashPoW) / chaincfg.ActiveNetParams.EthashEpochLength)

	// Recompute the digest and PoW values
	var (
		digest   []byte
		sealHash utils.Hash
	)

	// If fast-but-heavy PoW verification was requested, use an ethash dataset
	if fulldag {
		dataset := ethash.dataset(epoch, true)
		if dataset.generated() {
			digest, sealHash = hashimotoFull(dataset.dataset, header.ContentHash(), header.NonceExt)

			// Datasets are unmapped in a finalizer. Ensure that the dataset stays alive
			// until after the call to hashimotoFull so it's not unmapped while being used.
			runtime.KeepAlive(dataset)
		} else {
			// dataset not yet generated, don't hang, use a cache instead
			fulldag = false
		}
	}
	// If slow-but-light PoW verification was requested (or DAG not yet ready), use an ethash cache
	if !fulldag {
		cache := ethash.cache(epoch)

		size := datasetSize(epoch)
		if ethash.config.PowMode == ModeTest {
			size = 32 * 1024
		}
		digest, sealHash = hashimotoLight(size, cache.cache, header.ContentHash(), header.NonceExt)

		// Caches are unmapped in a finalizer. Ensure that the cache stays alive
		// until after the call to hashimotoLight so it's not unmapped while being used.
		runtime.KeepAlive(cache)
	}

	return sealHash, digest, nil
}

// Dataset() packages the dataset() function, and allow outsider can call this function.
// This is to allow the caller can have the dataset for specified block height, then can call other function, such as TrySeal.
// This function takes blockHeight as input, since the epoch and dataset mechanism are defined in ethash, and do not need to be known by the caller.
func (ethash *Ethash) Dataset(blockHeight int32) *dataset {
	//epoch := epoch(blockHeight)
	epoch := int((blockHeight - chaincfg.ActiveNetParams.BlockHeightEthashPoW) / chaincfg.ActiveNetParams.EthashEpochLength)
	return ethash.dataset(epoch, false)
}

// TrySeal() tris nonce to test whether the nonce generates a valid sealHash which lies in the scope specified by target,
// i.e., satisfying the consensus rules defined by VerifySeal().
// If the input nonce is valid, the block header will be updated, i.e., the MixDigest and NonceExt are set.
// To enable multiple calls on TrySeal() to use the same "dataset *dataset", this function takes "dataset *dataset" as an input,
// so that the miner can obtain a "dataset *dataset" by calling (ethash *Ethash) Dataset(blockHeight int32) and then call this function.
// To avoid the unnecessary computation of contentHash, this function takes contentHash as input,
// which should be computed by the caller on the input "header *wire.BlockHeader".
// That is, it is the responsibility of the caller to provide the dataset, contentHash, and target, corresponding to the input block header,
// and header *wire.BlockHeader is actually responsible to take the MixDigest and NonceExt out (if they are valid).
// To mine, TrySeal() will be called highly frequently.
// TrySeal() just provides a reference implementation for mining, i.e., finding valid EthashPoW solutions.
// Note that, the consensus rules are defined by VerifySeal(), and any nonce, whatever how it is found, is fine, as long as it satisfies the rules defined by VerifySeal().
func TrySeal(dataset *dataset, contentHash utils.Hash, nonceExt uint64, target *big.Int, header *wire.BlockHeader) bool {

	// Compute the PoW value of this nonce
	digest, sealHash := hashimotoFull(dataset.dataset, contentHash, nonceExt)

	//if hashToBig(sealHash).Cmp(target) <= 0 {
	//	header.NonceExt = nonceExt
	//
	//	header.MixDigest = utils.Hash{} // to make sure header.MixDigest != nil
	//	copy(header.MixDigest[:], digest)
	//
	//	return true
	//}

	//	Optimization: directly copy the codes of hashToBig() here, rather call the function hashToBig(),
	//	since TrySeal() will be called in very frequently in mining.
	//	This part codes must match with that in VerifySeal().
	for i := 0; i < utils.HashSize/2; i++ {
		sealHash[i], sealHash[utils.HashSize-1-i] = sealHash[utils.HashSize-1-i], sealHash[i]
	}
	if new(big.Int).SetBytes(sealHash[:]).Cmp(target) <= 0 {

		header.NonceExt = nonceExt

		header.MixDigest = utils.Hash{} // to make sure header.MixDigest != nil
		copy(header.MixDigest[:], digest)

		return true
	}

	return false
}

// There are two hashes in one TrySeal (actually in hashimoto()), one for naive hash, and one on input the mixDigest computed from DAG.
// While the later one takes much more time than the former one, we only count the later one for hash rate.
// Or, in other words, the hast times means the number of tried nonce.
const HashPerTrySeal = 1 //2

// SealHash() returns the sealHash of the input block header.
// The algorithm codes are consistent with the codes in hashimoto().
// This algorithm is used only where sealHash is needed independently, for example, for display.
func SealHash(header *wire.BlockHeader) utils.Hash {
	if header == nil || header.Version < wire.BlockVersionEthashPow {
		return utils.InvalidHash
	}

	// Combine contentHash + nonce into a 64 byte seed
	contentHash := header.ContentHash()

	seedTmp := make([]byte, utils.HashSize+8)
	copy(seedTmp, contentHash[:])
	binary.LittleEndian.PutUint64(seedTmp[utils.HashSize:], header.NonceExt)

	// we use the standard SHA3-512, rather than LegacyKeccak512
	seed := sha3.Sum512(seedTmp)

	return utils.ChainHash(append(seed[:], header.MixDigest[:]...))
}

// hashToBig converts a utils.Hash into a big.Int that can be used to perform math comparisons.
//
//	Note that the input is a utils.Hash ([32]byte) rather than a pointer,
//	the reverse in this function will not affect the content of the caller.
func hashToBig(hash utils.Hash) *big.Int {
	// As (hash Hash) String() returns the Hash as the hexadecimal string of the byte-reversed hash,
	// to make the big.Int value to be consistent with the displayed string, here also reverse it.
	//	This is also to be compatible with the exiting blocks.
	for i := 0; i < utils.HashSize/2; i++ {
		hash[i], hash[utils.HashSize-1-i] = hash[utils.HashSize-1-i], hash[i]
	}
	return new(big.Int).SetBytes(hash[:])
}

// PrepareDataset is used for preparing the dataset.
func (ethash *Ethash) PrepareDataset(epoch int) {
	ethash.dataset(epoch, true)
}

// PrepareCache is used for preparing the cache.
func (ethash *Ethash) PrepareCache(epoch int) {
	ethash.cache(epoch)
}
