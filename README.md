# Cloudflare DynDNS Updater (cfddns)

A Golang application that updates Cloudflare DNS records with your current public IP address.

## Features

- Updates both A (IPv4) and AAAA (IPv6) records
- Auto-detects public IP addresses if not provided
- Maintains existing proxy settings for each record
- Containerized for easy deployment on any platform
- Supports multiple architectures

## Installation

### Pre-built Binaries

Pre-built binaries for various platforms are available on the [GitHub Releases page](https://github.com/lhaig/cloudflare_dyndns/releases).

### Docker Image

Pre-built multi-architecture Docker images are available on Docker Hub:

```bash
docker pull lhaig/cfddns:latest
```

## Usage

### Command Line Arguments

```
  -zone-id string
        Cloudflare Zone ID (required)
  -api-token string
        Cloudflare API Token (required)
  -hostname string
        Hostname to update (required)
  -ip string
        IP Address (optional, will be auto-detected if not provided)
  -ipv6
        Enable IPv6 support (default true)
```

### Environment Variables

Instead of command line arguments, you can use these environment variables:

- `CLOUDFLARE_ZONE_ID`: Your Cloudflare Zone ID
- `CLOUDFLARE_API_TOKEN`: Your Cloudflare API Token
- `CLOUDFLARE_HOSTNAME`: The hostname to update
- `IP_ADDRESS`: Optional specific IP address to use

### Running Locally

```bash
# Run directly with Go
go run cmd/main.go -zone-id YOUR_DNS_ZONE_ID -api-token YOUR_API_TOKEN -hostname example.com

# Or build and run the binary
go build -o cfddns ./cmd/main.go
./cfddns -zone-id YOUR_DNS_ZONE_ID -api-token YOUR_API_TOKEN -hostname example.com
```

### Docker Container

Run the container:

```bash
docker run --rm lhaig/cfddns:latest -zone-id YOUR_DNS_ZONE_ID -api-token YOUR_DNS_API_TOKEN -hostname example.com
```

Or with environment variables:

```bash
docker run --rm \
  -e CLOUDFLARE_ZONE_ID=YOUR_ZONE_ID \
  -e CLOUDFLARE_API_TOKEN=YOUR_API_TOKEN \
  -e CLOUDFLARE_HOSTNAME=example.com \
  lhaig/cfddns:latest
```

## Development

### Building Locally

Build the binary:

```bash
make build
```

Build the Docker image:

```bash
make docker
```

Run tests:

```bash
make test
```

### Building Multi-architecture Images Locally

To build for multiple architectures using Docker BuildX:

```bash
make docker-buildx
```

### CI/CD Pipeline

This project uses GitHub Actions to automatically:

1. Build binaries for multiple platforms (Linux, Windows, macOS) and architectures (amd64, arm64, arm/v7)
2. Run tests to ensure code quality
3. Create GitHub releases with pre-built binaries when a tag is pushed
4. Build and push multi-architecture Docker images to Docker Hub

The workflow is triggered on:
- Pushes to the main branch
- Pull requests to the main branch
- Pushes of tags starting with 'v' (e.g., v1.0.0)

#### Creating a Release

To create a new release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will trigger the GitHub Actions workflow, which will build the binaries, create a GitHub release, and publish Docker images tagged with the version number.

## Automation

For automatic updates, you can run this container using a cron job or a scheduled task.

### Example Nomad Job

```hcl
job "cfddns" {
  datacenters = ["dc1"]
  type = "batch"

  periodic {
    cron = "*/15 * * * *"  # Run every 15 minutes
    prohibit_overlap = true
  }

  group "dyndns" {
    task "update" {
      driver = "docker"

      config {
        image = "lhaig/cfddns:latest"
      }

      env {
        CLOUDFLARE_ZONE_ID = "your-zone-id"
        CLOUDFLARE_API_TOKEN = "your-api-token"
        CLOUDFLARE_HOSTNAME = "your-hostname"
      }
    }
  }
}
```

## Return Values

The application will print one of the following values:

- `good`: The update was successful
- `nochg`: The IP address has not changed
- `badauth`: Authentication failed or another error occurred