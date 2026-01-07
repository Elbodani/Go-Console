package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/redis/go-redis/v9"
)

const (
	ValkeyAddr = "localhost:6379"
	ValkeyDB   = 0
)

func main() {
	var uploadDir, downloadDir, downloadTargetDir string
	upload := flag.Bool("u", false, "Upload files from a directory to Valkey")
	download := flag.Bool("d", false, "Download files from Valkey to a directory")
	flag.StringVar(&uploadDir, "upload-dir", "", "Directory to upload (use with -u)")
	flag.StringVar(&downloadDir, "download-key", "", "Valkey key for the directory (use with -d)")
	flag.StringVar(&downloadTargetDir, "target-dir", "", "Target directory for download (use with -d)")
	flag.Parse()

	rdb := redis.NewClient(&redis.Options{
		Addr: ValkeyAddr,
		DB:   ValkeyDB,
	})

	// Check the connection to Valkey
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Valkey: %v", err)
	}
	fmt.Println("Successfully connected to Valkey.")

	if *upload && uploadDir != "" {
		if err := uploadDirectory(ctx, rdb, uploadDir); err != nil {
			log.Fatalf("Upload failed: %v", err)
		}
	} else if *download && downloadDir != "" && downloadTargetDir != "" {
		if err := downloadDirectory(ctx, rdb, downloadDir, downloadTargetDir); err != nil {
			log.Fatalf("Download failed: %v", err)
		}
	} else {
		log.Println("Invalid arguments. Use -u for upload or -d for download.")
		flag.Usage()
		os.Exit(1)
	}

	// Close the connection
	rdb.Close()
}

func uploadDirectory(ctx context.Context, rdb *redis.Client, sourceDir string) error {
	fmt.Printf("Uploading files from '%s' to Valkey...\n", sourceDir)

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}
		key := filepath.Join(filepath.Base(sourceDir), relPath)
		key = strings.ReplaceAll(key, `\`, "/")

		err = rdb.Set(ctx, key, content, 0).Err() // 0 means no expiration
		if err != nil {
			return fmt.Errorf("failed to upload file %s with key %s: %w", path, key, err)
		}
		fmt.Printf("Uploaded: %s -> Key: %s\n", path, key)

		return nil
	})
}

func downloadDirectory(ctx context.Context, rdb *redis.Client, valkeyDirKey, targetDir string) error {
	fmt.Printf("Downloading files from Valkey directory key '%s' to '%s'...\n", valkeyDirKey, targetDir)

	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
	}

	pattern := valkeyDirKey + "/*"
	iter := rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		content, err := rdb.Get(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("failed to get value for key %s: %w", key, err)
		}

		localRelPath := strings.TrimPrefix(key, valkeyDirKey+"/")
		localFilePath := filepath.Join(targetDir, localRelPath)

		localDirPath := filepath.Dir(localFilePath)
		if err := os.MkdirAll(localDirPath, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create subdirectory %s: %w", localDirPath, err)
		}

		if err := os.WriteFile(localFilePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", localFilePath, err)
		}
		fmt.Printf("Downloaded: Key: %s -> %s\n", key, localFilePath)
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("error during scan iteration: %w", err)
	}

	return nil
}
