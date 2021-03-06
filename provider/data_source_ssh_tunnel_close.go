package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSSHTunnelClose() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSSHTunnelCloseRead,
		Schema:      map[string]*schema.Schema{},
	}
}

func dataSourceSSHTunnelCloseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	return diags
}
