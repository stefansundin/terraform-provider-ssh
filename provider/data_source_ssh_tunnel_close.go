package provider

import (
	"context"
	"log"
	"syscall"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSSHTunnelClose() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSSHTunnelCloseRead,
		Schema:      map[string]*schema.Schema{},
	}
}

func dataSourceSSHTunnelCloseRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	providerManager := m.(*SSHProviderManager)
	log.Printf("[DEBUG] Closing connections")
	for _, pid := range providerManager.TunnelProcessPIDs {
		if err := syscall.Kill(pid, syscall.SIGSTOP); err != nil {
			return diag.FromErr(err)
		}
	}
	return diags
}
