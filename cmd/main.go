package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lhaig/cloudflare_dyndns/internal"
)

func main() {
	// Parse command-line arguments
	zoneID := flag.String("zone-id", os.Getenv("CLOUDFLARE_ZONE_ID"), "Cloudflare Zone ID")
	apiToken := flag.String("api-token", os.Getenv("CLOUDFLARE_API_TOKEN"), "Cloudflare API Token")
	hostname := flag.String("hostname", os.Getenv("CLOUDFLARE_HOSTNAME"), "Hostname to update")
	ipAddr := flag.String("ip", os.Getenv("IP_ADDRESS"), "IP Address (optional, will be auto-detected if not provided)")
	enableIPv6 := flag.Bool("ipv6", true, "Enable IPv6 support")

	flag.Parse()

	// Check required parameters
	if *zoneID == "" || *apiToken == "" || *hostname == "" {
		fmt.Println("missing-credentials")
		fmt.Println("Error: zone-id, api-token, and hostname are required")
		flag.Usage()
		os.Exit(1)
	}

	// Create updater and run
	updater := internal.NewDNSUpdater(*zoneID, *apiToken, *hostname, *ipAddr, *enableIPv6)
	result, err := updater.Run()
	if err != nil {
		fmt.Println("authentication-error")
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println(result)
}
