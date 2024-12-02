package sync

import (
	"fmt"
	"os"
	"path/filepath"

	"gosync/internal/progress"
	"gosync/pkg/checksum"
	"gosync/pkg/utils"
	"gosync/internal/crypto"
)

// Manager handles file synchronization operations
type Manager struct {
	checksumCalc *checksum.Calculator
	blockSize    int64
	ignorePatterns []string
}

// NewManager creates a new sync manager
func NewManager(blockSize int64, ignorePatterns []string) *Manager {
	return &Manager{
		checksumCalc:    checksum.NewCalculator(blockSize),
		blockSize:       blockSize,
		ignorePatterns: ignorePatterns,
	}
}

// SyncDirectory synchronizes two directories with optional encryption
func (m *Manager) SyncDirectory(source, dest string, cryptoManager *crypto.Manager) error {
	// Get total size for progress tracking
	var totalSize int64
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !isSymlink(info.Mode()) {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error calculating total size: %w", err)
	}

	tracker := progress.NewTracker(totalSize)

	// Walk through source directory
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relativePath, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("error getting relative path: %w", err)
		}

		// Skip files matching ignore patterns
		for _, pattern := range m.ignorePatterns {
			match, err := filepath.Match(pattern, relativePath)
			if err != nil {
				return fmt.Errorf("error matching pattern: %w", err)
			}
			if match {
				return nil
			}
		}

		// Construct destination path
		destPath := filepath.Join(dest, relativePath)

		// Handle different file types
		mode := info.Mode()
		switch {
		case mode.IsDir():
			// Create directory
			if err := os.MkdirAll(destPath, mode.Perm()); err != nil {
				return fmt.Errorf("error creating directory %s: %w", destPath, err)
			}
			return nil

		case isSymlink(mode):
			// Read and recreate symlink
			link, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("error reading symlink %s: %w", path, err)
			}

			// Remove existing symlink if it exists
			_ = os.Remove(destPath)

			// Create parent directory
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("error creating parent directory for symlink: %w", err)
			}

			// Create new symlink
			if err := os.Symlink(link, destPath); err != nil {
				return fmt.Errorf("error creating symlink %s: %w", destPath, err)
			}
			return nil

		default:
			// Regular file
			// Ensure destination directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("error creating destination directory: %w", err)
			}

			// Sync the file with optional encryption
			if cryptoManager != nil {
				if err := cryptoManager.EncryptFile(path, destPath); err != nil {
					return fmt.Errorf("error encrypting file %s: %w", path, err)
				}
			} else {
				if err := utils.CopyFile(path, destPath); err != nil {
					return fmt.Errorf("error copying file %s: %w", path, err)
				}
			}

			// Update progress
			tracker.Update(info.Size())
			return nil
		}
	})
}

// isSymlink checks if the file mode indicates a symbolic link
func isSymlink(mode os.FileMode) bool {
	return mode&os.ModeSymlink != 0
}

// syncFile synchronizes a single file
func (m *Manager) syncFile(source, dest string, tracker *progress.Tracker) error {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	destInfo, err := os.Stat(dest)
	if err == nil {
		// File exists, check if it needs updating
		if sourceInfo.ModTime().Equal(destInfo.ModTime()) &&
			sourceInfo.Size() == destInfo.Size() {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	// Copy file with progress tracking
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	buffer := make([]byte, m.blockSize)
	for {
		n, err := sourceFile.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}

		if _, err := destFile.Write(buffer[:n]); err != nil {
			return err
		}

		tracker.Update(int64(n))
	}

	// Preserve modification time
	return os.Chtimes(dest, sourceInfo.ModTime(), sourceInfo.ModTime())
}
