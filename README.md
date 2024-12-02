# GoSync - CLI File Synchronization Tool

A powerful command-line file synchronization tool built in Go that provides secure, efficient, and real-time file synchronization across directories and machines.

## Features

### 1. File Watching
- Real-time monitoring of directory changes using `fsnotify`
- Detection of file creation, modification, and deletion events
- Configurable watch paths and ignore patterns
- Efficient event debouncing to prevent duplicate syncs

### 2. Differential Sync
- Efficient file comparison using checksums
- Block-level file diffing for large files
- Smart sync that only transfers changed portions
- Conflict detection and resolution
- Resume capability for interrupted transfers

### 3. Encryption
- AES-256 encryption for file transfers
- TLS for secure communication between nodes
- Key management and rotation
- Optional at-rest encryption for synchronized files

### 4. Progress Tracking
- Real-time transfer progress display
- Transfer speed and ETA calculations
- Detailed sync statistics
- Activity logging with different verbosity levels

### 5. Cross-platform Support
- Works on Linux, macOS, and Windows
- Handles different path separators
- Preserves file permissions and attributes
- Configurable for different filesystem quirks

## Project Structure

```
gosync/
├── cmd/
│   └── gosync/          # Main application entry point
├── internal/
│   ├── watcher/         # File system watching
│   ├── sync/           # Sync logic and diffing
│   ├── crypto/         # Encryption handling
│   ├── progress/       # Progress tracking
│   └── platform/       # Platform-specific code
├── pkg/
│   ├── checksum/       # Checksum calculation
│   ├── config/         # Configuration handling
│   └── utils/          # Common utilities
└── config/             # Configuration files
```

## Getting Started

### Prerequisites
- Go 1.20 or higher
- Git

### Installation
```bash
go install github.com/yourusername/gosync@latest
```

### Basic Usage
```bash
# Start syncing two directories locally
gosync sync /path/to/source /path/to/destination

# Watch a directory for changes
gosync watch /path/to/watch

# Create an encrypted sync
gosync sync --encrypt /source /destination

# Sync to a remote machine
gosync sync --remote /local/path /remote/path

# You can also use encryption
gosync sync --remote --encrypt ./local/files /remote/backup
```


## Configuration
Create a `config.yaml` file:
```yaml
sync:
  ignore_patterns:
    - "*.tmp"
    - ".git/"
  block_size: 4096
  compression: true

encryption:
  enabled: true
  key_file: "~/.gosync/keys/master.key"

watch:
  debounce_ms: 100
  recursive: true

# Remote sync configuration (optional)
remote:
  host: "remote-server.com"     # Remote server hostname or IP
  port: 22                      # SSH port (default: 22)
  username: "user"              # SSH username
  password: ""                  # SSH password (optional)
  key_file: "~/.ssh/id_rsa"    # SSH private key file (optional)
```

### Remote Sync
To sync files with a remote machine:

1. Configure the remote section in your `config.yaml`
2. Use the `--remote` flag with the sync command
3. The destination path will be relative to the remote user's home directory

Example:
```bash
# Sync local directory to remote server
gosync sync --remote ./local/files /remote/backup

# Sync with encryption
gosync sync --remote --encrypt ./local/files /remote/backup
```

You can authenticate using either a password or an SSH key file. If both are provided, the SSH key takes precedence.
