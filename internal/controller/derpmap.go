package controller

import (
	"hash/fnv"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/linkbit/linkbit/internal/models"
	"tailscale.com/tailcfg"
)

func buildDERPMap(relays []models.RelayNode) *tailcfg.DERPMap {
	regionsByCode := make(map[string]*tailcfg.DERPRegion)
	for _, relay := range relays {
		if relay.Status == models.RelayStatusUnhealthy {
			continue
		}
		node, ok := relayToDERPNode(relay)
		if !ok {
			continue
		}
		code := relay.Region
		if code == "" {
			code = "default"
		}
		region, ok := regionsByCode[code]
		if !ok {
			region = &tailcfg.DERPRegion{
				RegionID:   stableRegionID(code),
				RegionCode: code,
				RegionName: code,
			}
			regionsByCode[code] = region
		}
		node.RegionID = region.RegionID
		region.Nodes = append(region.Nodes, node)
	}

	regions := make(map[int]*tailcfg.DERPRegion, len(regionsByCode))
	for _, region := range regionsByCode {
		regions[region.RegionID] = region
	}
	return &tailcfg.DERPMap{
		Regions:            regions,
		OmitDefaultRegions: true,
	}
}

func relayToDERPNode(relay models.RelayNode) (*tailcfg.DERPNode, bool) {
	parsed, err := url.Parse(relay.PublicURL)
	if err != nil || parsed.Host == "" {
		return nil, false
	}
	host := parsed.Hostname()
	if host == "" {
		return nil, false
	}
	port := 443
	if parsed.Port() != "" {
		parsedPort, err := strconv.Atoi(parsed.Port())
		if err == nil {
			port = parsedPort
		}
	} else if parsed.Scheme == "http" {
		port = 80
	}

	node := &tailcfg.DERPNode{
		Name:     sanitizeDERPName(relay.ID),
		HostName: host,
		DERPPort: port,
		STUNPort: -1,
		IPv4:     relay.IPv4,
		IPv6:     relay.IPv6,
	}
	if parsed.Scheme == "http" {
		node.InsecureForTests = true
	}
	if ip := net.ParseIP(host); ip != nil {
		node.IPv4 = host
		if ip.To4() == nil {
			node.IPv4 = "none"
			node.IPv6 = host
		}
	}
	return node, true
}

func stableRegionID(region string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(region))
	return 900 + int(h.Sum32()%100)
}

func sanitizeDERPName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "relay"
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "relay"
	}
	return builder.String()
}
