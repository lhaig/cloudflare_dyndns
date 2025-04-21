# Cloudflare DynDNS Updater (cfddns)

A Golang application that updates Cloudflare DNS records with your current public IP address. This is a Go implementation of the bash script for updating Cloudflare DNS records.

## Features

- Updates both A (IPv4) and AAAA (IPv6) records
- Auto-detects public IP addresses if not provided
- Maintains existing proxy settings for each record
- Containerized for easy deployment on any platform
- Supports multiple architectures

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

Build the Docker image:

```bash
docker build -t cfddns .
```

Run the container:

```bash
docker run --rm cfddns -zone-id YOUR_DNS_ZONE_ID -api-token YOUR_DNS_API_TOKEN -hostname example.com
```

Or with environment variables:

```bash
docker run --rm \
  -e CLOUDFLARE_ZONE_ID=YOUR_ZONE_ID \
  -e CLOUDFLARE_API_TOKEN=YOUR_API_TOKEN \
  -e CLOUDFLARE_HOSTNAME=example.com \
  cfddns
```

### Building Multi-architecture Images

To build for multiple architectures using Docker BuildX:

```bash
# Set up buildx builder instance
docker buildx create --name mybuilder --use

# Build and push the multi-architecture image
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t yourusername/cfddns:latest \
  --push .
```

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
        image = "yourusername/cfddns:latest"
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