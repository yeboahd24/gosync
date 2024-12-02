package checksum

import (
	"crypto/sha256"
	"io"
	"os"
)

// Calculator handles file checksum operations
type Calculator struct {
	blockSize int64
}

// NewCalculator creates a new checksum calculator with specified block size
func NewCalculator(blockSize int64) *Calculator {
	return &Calculator{
		blockSize: blockSize,
	}
}

// CalculateFileChecksum computes the SHA-256 checksum of an entire file
func (c *Calculator) CalculateFileChecksum(filepath string) ([]byte, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// CalculateBlockChecksum computes checksums for each block in a file
func (c *Calculator) CalculateBlockChecksum(filepath string) (map[int64][]byte, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	checksums := make(map[int64][]byte)
	buffer := make([]byte, c.blockSize)
	
	for blockNum := int64(0); ; blockNum++ {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		hash := sha256.New()
		hash.Write(buffer[:n])
		checksums[blockNum] = hash.Sum(nil)
	}

	return checksums, nil
}
