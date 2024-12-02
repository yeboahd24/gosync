package platform

import (
	"os"
	"runtime"
)

// GetPathSeparator returns the platform-specific path separator
func GetPathSeparator() string {
	return string(os.PathSeparator)
}

// GetLineEnding returns the platform-specific line ending
func GetLineEnding() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// GetDefaultConfigPath returns the platform-specific default config path
func GetDefaultConfigPath() string {
	if IsWindows() {
		return os.Getenv("APPDATA") + GetPathSeparator() + "gosync" + GetPathSeparator() + "config.yaml"
	}
	return os.Getenv("HOME") + GetPathSeparator() + ".config" + GetPathSeparator() + "gosync" + GetPathSeparator() + "config.yaml"
}
