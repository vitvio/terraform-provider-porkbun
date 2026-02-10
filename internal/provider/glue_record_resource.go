package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	porkbun "github.com/vitvio/terraform-provider-porkbun/internal/client"
	"github.com/vitvio/terraform-provider-porkbun/internal/consts"
)

var (
	_ resource.Resource                = &GlueRecordResource{}
	_ resource.ResourceWithImportState = &GlueRecordResource{}
)

type GlueRecordResource struct {
	client *porkbun.Client
}

func NewGlueRecordResource() resource.Resource {
	return &GlueRecordResource{}
}

func (r *GlueRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_glue_record"
}

func (r *GlueRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage glue records (nameserver IP addresses) for your domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the glue record (format: domain:subdomain).",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The FQDN of the domain.",
				Required:            true,
			},
			"subdomain": schema.StringAttribute{
				MarkdownDescription: "The subdomain of the glue record (e.g., 'ns1').",
				Required:            true,
			},
			"ips": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "The IP addresses for the glue record.",
				Required:            true,
			},
		},
	}
}

type GlueRecordResourceModel struct {
	ID        types.String   `tfsdk:"id"`
	Domain    types.String   `tfsdk:"domain"`
	Subdomain types.String   `tfsdk:"subdomain"`
	IPs       []types.String `tfsdk:"ips"`
}

func (r *GlueRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*porkbun.Client)
	if !ok {
		resp.Diagnostics.AddError(
			consts.ErrUnexpectedResourceConfigureType,
			fmt.Sprintf("Expected *porkbun.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *GlueRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GlueRecordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ips := make([]string, len(data.IPs))
	for i, tfIP := range data.IPs {
		ips[i] = tfIP.ValueString()
	}

	err := r.client.CreateGlueRecord(ctx, data.Domain.ValueString(), data.Subdomain.ValueString(), ips)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create glue record", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.Domain.ValueString(), data.Subdomain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GlueRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GlueRecordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	records, err := r.client.GetGlueRecords(ctx, data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read glue records", err.Error())
		return
	}

	// Find the specific record
	found := false
	expectedHost := fmt.Sprintf("%s.%s", data.Subdomain.ValueString(), data.Domain.ValueString())

	for _, rec := range records {
		// rec.Subdomain from GetGlueRecords is actually the full hostname (e.g. ns1.example.com)
		// We can check exact match or if it ends with domain?
		// The API implementation I wrote: `records = append(records, GlueRecord{ ..., Subdomain: hostName })`
		// where hostName comes from the [0] element of the array.

		if rec.Subdomain == expectedHost {
			found = true
			tfIPs := make([]types.String, len(rec.IPs))
			for i, ip := range rec.IPs {
				tfIPs[i] = types.StringValue(ip)
			}
			data.IPs = tfIPs
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.Domain.ValueString(), data.Subdomain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GlueRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GlueRecordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ips := make([]string, len(data.IPs))
	for i, tfIP := range data.IPs {
		ips[i] = tfIP.ValueString()
	}

	// Porkbun separates UpdateGlueRecord from CreateGlueRecord, but functionally they just replace the IPs.
	err := r.client.UpdateGlueRecord(ctx, data.Domain.ValueString(), data.Subdomain.ValueString(), ips)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update glue record", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.Domain.ValueString(), data.Subdomain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GlueRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GlueRecordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteGlueRecord(ctx, data.Domain.ValueString(), data.Subdomain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete glue record", err.Error())
		return
	}
}

func (r *GlueRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: domain:subdomain. Got: %q", req.ID),
		)
		return
	}

	// Manually set attributes from the ID parts because the Read method relies on
	// "domain" and "subdomain" being present in the state to function correctly.
	// Standard ImportStatePassthroughID would not populate these fields effectively
	// for the Read operation to follow.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subdomain"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
