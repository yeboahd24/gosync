package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gosync/internal/crypto"
	"gosync/internal/network"
	"gosync/internal/platform"
	"gosync/internal/sync"
	"gosync/internal/watcher"
	"gosync/pkg/config"
)

func printUsage() {
	fmt.Printf(`GoSync - A file synchronization tool

Usage:
  gosync [command] [options] [arguments]

Commands:
  sync   Synchronize files from source to destination
         gosync sync [options] <source> <dest>
         
         Options:
           -encrypt    Enable encryption (requires config with key file)
           -compress   Enable compression (default: true)
           -remote     Sync to remote host (requires remote config)

  watch  Watch a directory for changes and sync automatically
         gosync watch [options] <directory>
         
         Options:
           -recursive  Watch directories recursively (default: true)
           -debounce   Debounce time in milliseconds (default: 100)

Examples:
  gosync sync ./source ./backup
  gosync sync -encrypt ./source ./backup
  gosync sync -remote ./source /remote/backup
  gosync watch -recursive ./directory

For more information, visit: https://github.com/yourusername/gosync
`)
}

func main() {
	// Define subcommands
	syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
	watchCmd := flag.NewFlagSet("watch", flag.ExitOnError)

	// Sync command flags
	syncEncrypt := syncCmd.Bool("encrypt", false, "Enable encryption for sync")
	syncCompress := syncCmd.Bool("compress", true, "Enable compression")
	syncRemote := syncCmd.Bool("remote", false, "Sync to remote host (requires remote config)")

	// Watch command flags
	watchRecursive := watchCmd.Bool("recursive", true, "Watch directories recursively")
	watchDebounce := watchCmd.Int("debounce", 100, "Debounce time in milliseconds")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Check for help flags
	if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		printUsage()
		os.Exit(0)
	}

	// Load configuration
	var cfg *config.Config
	var err error

	switch os.Args[1] {
	case "sync":
		syncCmd.Parse(os.Args[2:])
		if syncCmd.NArg() != 2 {
			fmt.Println("Error: sync requires source and destination paths")
			fmt.Println("\nUsage: gosync sync [options] <source> <dest>")
			syncCmd.PrintDefaults()
			os.Exit(1)
		}
		cfg, err = loadConfig("")
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		handleSync(syncCmd.Arg(0), syncCmd.Arg(1), cfg, *syncEncrypt, *syncCompress, *syncRemote)

	case "watch":
		watchCmd.Parse(os.Args[2:])
		if watchCmd.NArg() != 1 {
			fmt.Println("Error: watch requires a directory path")
			fmt.Println("\nUsage: gosync watch [options] <directory>")
			watchCmd.PrintDefaults()
			os.Exit(1)
		}
		cfg, err = loadConfig("")
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		handleWatch(watchCmd.Arg(0), cfg, *watchRecursive, *watchDebounce)

	default:
		fmt.Printf("%q is not valid command.\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func loadConfig(configPath string) (*config.Config, error) {
	// If no config path specified, try different locations
	if configPath == "" {
		// Try current directory first
		if _, err := os.Stat("config.yaml"); err == nil {
			return config.LoadConfig("config.yaml")
		}

		// Try default system config location
		defaultPath := platform.GetDefaultConfigPath()
		if _, err := os.Stat(defaultPath); err == nil {
			return config.LoadConfig(defaultPath)
		}

		// Create default config
		defaultConfig := &config.Config{
			Sync: config.SyncConfig{
				BlockSize:      4096,
				IgnorePatterns: []string{".git/", "*.tmp", "*.swp"},
				Compression:    true,
			},
			Encryption: config.EncryptionConfig{
				Enabled: false,
				KeyFile: "",
			},
			Watch: config.WatchConfig{
				DebounceMs: 100,
				Recursive:  true,
			},
			Remote: config.RemoteConfig{
				Host:     "",
				Port:     0,
				Username: "",
				Password: "",
				KeyFile:  "",
			},
		}

		// Try to save in system location first
		defaultPath = platform.GetDefaultConfigPath()
		configDir := filepath.Dir(defaultPath)
		if err := os.MkdirAll(configDir, 0755); err == nil {
			if err := config.SaveConfig(defaultConfig, defaultPath); err == nil {
				return defaultConfig, nil
			}
		}

		// If system location fails, try current directory
		if err := config.SaveConfig(defaultConfig, "config.yaml"); err == nil {
			return defaultConfig, nil
		}

		// If both fail, just return the default config in memory
		return defaultConfig, nil
	}
	return config.LoadConfig(configPath)
}

func handleSync(source, dest string, cfg *config.Config, encrypt, compress, remote bool) {
	source, err := filepath.Abs(source)
	if err != nil {
		log.Fatalf("Invalid source path: %v", err)
	}

	if remote {
		// Check remote configuration
		if cfg.Remote.Host == "" {
			log.Fatal("Remote sync requires host configuration in config file")
		}
		if cfg.Remote.Port == 0 {
			cfg.Remote.Port = 22 // Default SSH port
		}

		// Convert destination path to use forward slashes for remote systems
		dest = filepath.ToSlash(dest)

		fmt.Printf("Syncing from %s to %s@%s:%s\n", source, cfg.Remote.Username, cfg.Remote.Host, dest)
		fmt.Printf("Encryption: %v, Compression: %v\n", encrypt, compress)

		// Initialize remote sync
		remoteSync, err := network.NewRemoteSync(network.RemoteConfig{
			Host:     cfg.Remote.Host,
			Port:     cfg.Remote.Port,
			Username: cfg.Remote.Username,
			Password: cfg.Remote.Password,
			KeyFile:  cfg.Remote.KeyFile,
		}, dest)
		if err != nil {
			log.Fatalf("Error initializing remote sync: %v", err)
		}
		defer remoteSync.Close()

		// Sync to remote
		if err := remoteSync.SyncToRemote(source); err != nil {
			log.Fatalf("Error during remote sync: %v", err)
		}
	} else {
		dest, err = filepath.Abs(dest)
		if err != nil {
			log.Fatalf("Invalid destination path: %v", err)
		}

		fmt.Printf("Syncing from %s to %s\n", source, dest)
		fmt.Printf("Encryption: %v, Compression: %v\n", encrypt, compress)

		// Initialize sync manager
		syncManager := sync.NewManager(cfg.Sync.BlockSize, cfg.Sync.IgnorePatterns)

		// Initialize crypto manager if encryption is enabled
		var cryptoManager *crypto.Manager
		if encrypt {
			cryptoManager, err = crypto.NewManager(cfg.Encryption.KeyFile)
			if err != nil {
				log.Fatalf("Error initializing crypto manager: %v", err)
			}
		}

		// Perform sync
		if err := syncManager.SyncDirectory(source, dest, cryptoManager); err != nil {
			log.Fatalf("Error during sync: %v", err)
		}
	}

	fmt.Println("Sync completed successfully")
}

func handleWatch(dir string, cfg *config.Config, recursive bool, debounce int) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("Invalid directory path: %v", err)
	}

	fmt.Printf("Watching directory: %s\n", dir)
	fmt.Printf("Recursive: %v, Debounce: %dms\n", recursive, debounce)

	// Initialize watcher
	w, err := watcher.NewWatcher(debounce)
	if err != nil {
		log.Fatalf("Error creating watcher: %v", err)
	}
	defer w.Close()

	// Start watching
	if err := w.Watch(dir, recursive); err != nil {
		log.Fatalf("Error starting watcher: %v", err)
	}

	// Handle events
	fmt.Println("Watching for changes. Press Ctrl+C to stop.")
	for {
		select {
		case event := <-w.Events():
			fmt.Printf("Event: %s - %s\n", event.Operation, event.Path)
		case err := <-w.Errors():
			log.Printf("Error: %v\n", err)
		}
	}
}
