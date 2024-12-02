package network

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// RemoteConfig holds the configuration for remote connection
type RemoteConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	KeyFile  string
}

// RemoteSync handles remote file synchronization
type RemoteSync struct {
	client     *sftp.Client
	sshClient  *ssh.Client
	remoteBase string
}

// NewRemoteSync creates a new remote sync handler
func NewRemoteSync(config RemoteConfig, remoteBase string) (*RemoteSync, error) {
	var authMethods []ssh.AuthMethod

	if config.Password != "" {
		authMethods = append(authMethods, ssh.Password(config.Password))
	}

	if config.KeyFile != "" {
		key, err := os.ReadFile(config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("unable to parse private key: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication methods provided")
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Add proper host key verification
	}

	// Connect to remote host
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote host: %w", err)
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	return &RemoteSync{
		client:     sftpClient,
		sshClient:  sshClient,
		remoteBase: remoteBase,
	}, nil
}

// Close closes the remote connection
func (r *RemoteSync) Close() error {
	if err := r.client.Close(); err != nil {
		return err
	}
	return r.sshClient.Close()
}

// CopyToRemote copies a file to the remote host
func (r *RemoteSync) CopyToRemote(localPath, remotePath string) error {
	// Open local file
	local, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer local.Close()

	// Ensure remote directory exists
	remoteDir := filepath.Dir(remotePath)
	if err := r.mkdirAll(remoteDir); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Create remote file
	remote, err := r.client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer remote.Close()

	// Copy file contents
	if _, err := io.Copy(remote, local); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Copy file mode
	info, err := local.Stat()
	if err != nil {
		return fmt.Errorf("failed to get local file info: %w", err)
	}

	if err := r.client.Chmod(remotePath, info.Mode()); err != nil {
		return fmt.Errorf("failed to set remote file permissions: %w", err)
	}

	return nil
}

// mkdirAll creates a directory and all parent directories on the remote host
func (r *RemoteSync) mkdirAll(path string) error {
	if path == "" {
		return nil
	}

	// Split path into components
	components := strings.Split(filepath.Clean(path), string(filepath.Separator))
	current := ""

	// Create each component
	for _, component := range components {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		r.client.Mkdir(current) // Ignore errors as directory might already exist
	}

	return nil
}

// SyncToRemote synchronizes a local directory to a remote directory
func (r *RemoteSync) SyncToRemote(localPath string) error {
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Construct remote path
		remotePath := filepath.Join(r.remoteBase, relPath)

		if info.IsDir() {
			// Create directory on remote
			return r.mkdirAll(remotePath)
		} else if info.Mode()&os.ModeSymlink != 0 {
			// Handle symlinks
			link, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read symlink: %w", err)
			}

			// Remove existing symlink if it exists
			r.client.Remove(remotePath)

			// Create new symlink
			if err := r.client.Symlink(link, remotePath); err != nil {
				return fmt.Errorf("failed to create remote symlink: %w", err)
			}
		} else {
			// Copy regular file
			if err := r.CopyToRemote(path, remotePath); err != nil {
				return err
			}
		}

		return nil
	})
}
