package provider

import (
	"context"
	"log"

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
	for _, listener := range providerManager.Listeners {
		defer (*listener).Close()
	}
	return diags
}
