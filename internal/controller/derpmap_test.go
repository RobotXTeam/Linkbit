package controller

import (
	"testing"

	"github.com/linkbit/linkbit/internal/models"
)

func TestBuildDERPMap(t *testing.T) {
	derpMap := buildDERPMap([]models.RelayNode{
		{
			ID:        "relay-1",
			Region:    "home",
			PublicURL: "https://relay.example.com:8443",
			Status:    models.RelayStatusHealthy,
		},
	})

	if len(derpMap.Regions) != 1 {
		t.Fatalf("regions = %d, want 1", len(derpMap.Regions))
	}
	for _, region := range derpMap.Regions {
		if region.RegionCode != "home" {
			t.Fatalf("region code = %s", region.RegionCode)
		}
		if len(region.Nodes) != 1 {
			t.Fatalf("nodes = %d, want 1", len(region.Nodes))
		}
		if region.Nodes[0].HostName != "relay.example.com" || region.Nodes[0].DERPPort != 8443 {
			t.Fatalf("unexpected node: %+v", region.Nodes[0])
		}
	}
}

func TestRelayToDERPNodeIPLiteral(t *testing.T) {
	node, ok := relayToDERPNode(models.RelayNode{
		ID:        "relay-ip",
		Region:    "home",
		PublicURL: "http://203.0.113.10:8080",
		Status:    models.RelayStatusHealthy,
	})
	if !ok {
		t.Fatal("relayToDERPNode() returned false")
	}
	if node.IPv4 != "203.0.113.10" || !node.InsecureForTests || node.DERPPort != 8080 {
		t.Fatalf("unexpected node: %+v", node)
	}
}
