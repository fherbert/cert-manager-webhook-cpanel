package main

import (
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"github.com/fherbert/cert-manager-webhook-cpanel/internal/solver"
	"k8s.io/klog/v2"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		klog.Fatal("GROUP_NAME must be specified")
	}

	cmd.RunWebhookServer(GroupName,
		&solver.CPanelDNSProviderSolver{},
	)
}
