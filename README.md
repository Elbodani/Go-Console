# Go-Console
Go console application for uploading/downloading files to/from Valkey database

 
## Description

This tool allows you to:
- Upload all files from a directory to Valkey database
- Download files from Valkey database to a local directory
- Store files as key-value pairs where key = `directory:filename` and value = file contents

## Requirements

- **Go** (version 1.20 or higher)
  - Download from: https://go.dev/dl/
  - Verify installation: `go version`

- **Docker Desktop for Windows**
  - Download from: https://www.docker.com/products/docker-desktop/
  - Verify installation: `docker --version`

- **Valkey** (running in Docker container)
  - No separate installation needed - runs via Docker

