# Gemini Enterprise CLI (Go)

A powerful Go CLI tool for managing Google Gemini Enterprise (formerly Google Cloud Agentspace) resources with a gcloud-style interface.

## Overview

This CLI provides comprehensive management of Gemini Enterprise resources including:
- **Engines** (AI apps) - Search engines and conversational AI applications
- **Data Stores** - Document storage and processing systems
- **Documents** - Individual documents within data stores

## Features

- **gcloud-style commands** for familiar developer experience
- **Complete resource management** (engines, data stores, documents)
- **Multiple authentication methods** (user credentials + service accounts)
- **Regional endpoint support** (global, us, eu)
- **Rich output formats** (table, JSON, YAML)
- **Automation-friendly** with comprehensive scripting support
- **Cross-platform** Go binary for easy deployment

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/vb140772/gemctl-go.git
cd gemctl-go

# Build the binary
go build -o gemctl

# Or install globally
go install
```

### Authentication Setup

```bash
# Option A: User credentials (default)
gcloud auth login

# Option B: Service account
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
```

### Basic Usage

```bash
# List engines
./gemctl engines list --project-id=my-project --location=us

# List data stores
./gemctl data-stores list --project-id=my-project --location=us

# Describe an engine
./gemctl engines describe my-engine --project-id=my-project --location=us

# Create a search engine
./gemctl engines create my-engine "My Search Engine" datastore1 datastore2

# Create data store from GCS
./gemctl data-stores create-from-gcs my-store "My Store" gs://my-bucket/docs/*
```

## Command Reference

### Global Options

| Option | Description | Default |
|--------|-------------|---------|
| `--project-id` | Google Cloud project ID | From gcloud config or $GOOGLE_CLOUD_PROJECT |
| `--location` | Region (us, us-central1, global) | From $AGENTSPACE_LOCATION or `us` |
| `--format` | Output format (table, json, yaml) | `table` |
| `--collection` | Collection ID | `default_collection` |
| `--use-service-account` | Use ADC instead of user credentials | User credentials |

### Engines Commands

#### `engines list`
List all engines in a project.

```bash
gemctl engines list [--project-id PROJECT_ID] [--location LOCATION] [--format FORMAT]
```

**Examples:**
```bash
# Table format (default)
gemctl engines list

# With explicit flags
gemctl engines list --project-id=my-project --location=us

# JSON format
gemctl engines list --project-id=my-project --location=us --format=json
```

#### `engines describe`
Describe a specific engine.

```bash
gemctl engines describe ENGINE_ID [--project-id PROJECT_ID] [--location LOCATION] [--format FORMAT] [--full]
```

**Examples:**
```bash
# Basic describe
gemctl engines describe my-engine

# Full configuration with data stores
gemctl engines describe my-engine --full

# JSON format
gemctl engines describe my-engine --format=json
```

#### `engines create`
Create a search engine connected to data stores.

```bash
gemctl engines create ENGINE_ID DISPLAY_NAME [DATA_STORE_IDS...] [--search-tier TIER]
```

**Examples:**
```bash
# Create engine with data stores
gemctl engines create my-engine "My Search Engine" datastore1 datastore2

# Create enterprise tier engine
gemctl engines create my-engine "My Search Engine" datastore1 --search-tier=SEARCH_TIER_ENTERPRISE

# Create engine without data stores
gemctl engines create my-engine "My Search Engine"
```

#### `engines delete`
Delete a search engine.

```bash
gemctl engines delete ENGINE_ID [--force]
```

**Examples:**
```bash
# Delete with confirmation
gemctl engines delete my-engine

# Delete without confirmation
gemctl engines delete my-engine --force
```

### Data Stores Commands

#### `data-stores list`
List all data stores in a project.

```bash
gemctl data-stores list [--project-id PROJECT_ID] [--location LOCATION] [--format FORMAT]
```

#### `data-stores describe`
Describe a specific data store.

```bash
gemctl data-stores describe DATA_STORE_ID [--project-id PROJECT_ID] [--location LOCATION] [--format FORMAT]
```

#### `data-stores create-from-gcs`
Create a data store and import data from GCS bucket.

```bash
gemctl data-stores create-from-gcs DATA_STORE_ID DISPLAY_NAME GCS_URI [--data-schema SCHEMA] [--reconciliation-mode MODE]
```

**Examples:**
```bash
# Create from PDF files
gemctl data-stores create-from-gcs my-docs "My Documents" gs://my-bucket/docs/*.pdf

# Create from CSV file
gemctl data-stores create-from-gcs my-data "My Data" gs://my-bucket/data.csv --data-schema=csv

# Full reconciliation mode
gemctl data-stores create-from-gcs my-store "My Store" gs://my-bucket/* --reconciliation-mode=FULL
```

#### `data-stores list-documents`
List documents in a data store.

```bash
gemctl data-stores list-documents DATA_STORE_ID [--branch BRANCH] [--format FORMAT]
```

#### `data-stores delete`
Delete a data store.

```bash
gemctl data-stores delete DATA_STORE_ID [--force]
```

## Authentication

### Method 1: User Credentials (Default)
Uses `gcloud auth print-access-token` - best for interactive use and development.

```bash
# Login with your Google account
gcloud auth login

# Use CLI (no additional flags needed)
gemctl engines list --project-id=my-project
```

### Method 2: Service Account (Recommended for automation)
Uses Application Default Credentials (ADC) - best for CI/CD and automated scripts.

```bash
# Option A: Service account key file
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
gemctl engines list --project-id=my-project --use-service-account

# Option B: Application default credentials
gcloud auth application-default login
gemctl engines list --project-id=my-project --use-service-account
```

### Required IAM Permissions

**For Read Operations:**
- `discoveryengine.collections.list`
- `discoveryengine.engines.list`
- `discoveryengine.engines.get`
- `discoveryengine.dataStores.list`
- `discoveryengine.dataStores.get`
- `discoveryengine.documents.list`

**For Create Operations:**
- `discoveryengine.dataStores.create`
- `discoveryengine.documents.import`
- `storage.objects.list` (for GCS bucket access)

**Predefined Roles:**
- `roles/discoveryengine.viewer` (for read-only operations)
- `roles/discoveryengine.admin` (for full access including create operations)

## Configuration

### Environment Variables

```bash
# Set defaults for convenience
export GOOGLE_CLOUD_PROJECT=my-project
export AGENTSPACE_LOCATION=us-central1

# Now run without flags
gemctl engines list
```

### Regional Endpoints

The CLI automatically uses the correct regional endpoint based on `--location`:

| Location | API Endpoint |
|----------|-------------|
| `global` | `https://discoveryengine.googleapis.com/v1` |
| `us`, `us-central1` | `https://us-discoveryengine.googleapis.com/v1` (default) |
| `eu`, `eu-west1` | `https://eu-discoveryengine.googleapis.com/v1` |

## Output Formats

### Table (default)
Human-readable columnar display with summary statistics.

### JSON
Machine-readable format suitable for automation and parsing with `jq`.

### YAML
Configuration-friendly format for deployment scripts.

## Usage Examples

### 1. List All Resources

```bash
# List engines
gemctl engines list --project-id=my-project --location=us

# List data stores
gemctl data-stores list --project-id=my-project --location=us

# With service account
gemctl engines list --project-id=my-project --location=us --use-service-account
```

### 2. Export Configuration for Deployment

```bash
# Export full engine configuration
gemctl engines describe my-engine --project-id=my-project --location=us --full > engine-config.json

# With service account
gemctl engines describe my-engine --project-id=my-project --location=us --use-service-account --full > engine-config.json
```

### 3. Automation/Scripting

```bash
# Get JSON output for parsing
ENGINE_COUNT=$(gemctl engines list --project-id=my-project --location=us --format=json | jq 'length')
echo "Found $ENGINE_COUNT engines"

# List all engine names
gemctl engines list --project-id=my-project --location=us --format=json | jq -r '.[].displayName'
```

### 4. Backup All Engines

```bash
# Get all engine configurations
for engine in $(gemctl engines list --project-id=my-project --location=us --format=json | jq -r '.[].name | split("/") | .[-1]'); do
  gemctl engines describe "$engine" --project-id=my-project --location=us --full > "backup-${engine}.json"
done
```

## Development

### Project Structure

```
gemctl-go/
├── main.go                    # Entry point
├── go.mod                     # Go module definition
├── internal/
│   ├── cli/                   # CLI command definitions
│   │   ├── root.go           # Root command
│   │   ├── engines.go        # Engines command group
│   │   ├── engines_commands.go # Engine-specific commands
│   │   ├── datastores.go     # Data stores command group
│   │   ├── datastores_commands.go # Data store-specific commands
│   │   └── output.go         # Output formatting
│   └── client/                # Gemini Enterprise API client
│       ├── client.go         # Main client and configuration
│       ├── engines.go        # Engine operations
│       └── datastores.go     # Data store operations
└── README.md                  # This file
```

### Building

```bash
# Build for current platform
go build -o gemctl

# Build for specific platforms
GOOS=linux GOARCH=amd64 go build -o gemctl-linux-amd64
GOOS=windows GOARCH=amd64 go build -o gemctl-windows-amd64.exe
GOOS=darwin GOARCH=amd64 go build -o gemctl-darwin-amd64

# Build with GoReleaser (cross-platform)
make release-build
```

### Release Management

This project uses [GoReleaser](https://goreleaser.com) for automated releases:

```bash
# Test GoReleaser configuration
make release-test

# Build snapshot release
make release-build

# Create snapshot release (local)
make release

# Create full release (requires git tag)
make release-full
```

GoReleaser automatically:
- Builds for multiple platforms (Linux, Windows, macOS)
- Creates archives (tar.gz, zip)
- Generates checksums
- Creates GitHub releases
- Publishes to package managers

### Continuous Integration

The project includes GitHub Actions workflows:

- **CI** (`.github/workflows/ci.yml`): Runs tests and builds on multiple platforms
- **Release** (`.github/workflows/release.yml`): Automated releases when tags are pushed

To create a release:
```bash
git tag v1.0.0
git push origin v1.0.0
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test ./internal/client -v
```

## Troubleshooting

### API Not Enabled (403 Error)

**Symptom:** `403 Client Error: Forbidden`

**Solution:**
```bash
# Enable API in target project
gcloud services enable discoveryengine.googleapis.com --project=PROJECT_ID

# Wait 1-2 minutes for propagation
```

### Permission Denied

**Symptom:** `Permission 'discoveryengine.*.list' denied`

**Solution:**
```bash
# Grant permissions to service account
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member='serviceAccount:SA_EMAIL' \
  --role='roles/discoveryengine.viewer'

# Wait 30-60 seconds for IAM propagation
```

### Authentication Errors

**Symptom:** `DefaultCredentialsError: Your default credentials were not found`

**Solution:**
```bash
# Set up credentials
gcloud auth application-default login

# Or use service account
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"
```

## Comparison with Python Version

This Go implementation provides the same functionality as the Python version with these advantages:

- **Single binary** - No Python dependencies or virtual environments
- **Faster startup** - Compiled binary starts faster than Python interpreter
- **Cross-platform** - Easy deployment across different operating systems
- **Better performance** - Go's concurrency and performance benefits
- **Smaller footprint** - Single executable vs Python + dependencies

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

For questions or issues:
1. Check this README for documentation
2. Use `--help` at any command level
3. Review error messages carefully
4. Check Google Cloud Console to verify resources exist
5. Verify IAM permissions and API enablement

## Version History

- **v1.0.0** - Initial Go implementation
  - Complete CLI with gcloud-style commands
  - Engines and data stores management
  - Regional endpoint support
  - Table, JSON, and YAML output formats
  - User credentials and service account authentication
  - Comprehensive help system
