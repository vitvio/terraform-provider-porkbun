package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	porkbun "github.com/vitvio/terraform-provider-porkbun/internal/client"
	"github.com/vitvio/terraform-provider-porkbun/internal/consts"
)

var (
	_ resource.Resource                = &DNSSECRecordResource{}
	_ resource.ResourceWithImportState = &DNSSECRecordResource{}
)

type DNSSECRecordResource struct {
	client *porkbun.Client
}

func NewDNSSECRecordResource() resource.Resource {
	return &DNSSECRecordResource{}
}

func (r *DNSSECRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dnssec_record"
}

func (r *DNSSECRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage DNSSEC DS records at the registry for your domain. " +
			"Porkbun's API cannot update DNSSEC records in place, so changing any attribute replaces the record.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the DNSSEC record (format: domain:key_tag).",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The FQDN of the domain.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_tag": schema.StringAttribute{
				MarkdownDescription: "The key tag of the DS record (e.g., '5878').",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"algorithm": schema.StringAttribute{
				MarkdownDescription: "The DNSKEY algorithm number of the DS record (e.g., '13' for ECDSAP256SHA256).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"digest_type": schema.StringAttribute{
				MarkdownDescription: "The digest type number of the DS record (e.g., '2' for SHA-256).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"digest": schema.StringAttribute{
				MarkdownDescription: "The hex-encoded digest of the DS record.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

type DNSSECRecordResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Domain     types.String `tfsdk:"domain"`
	KeyTag     types.String `tfsdk:"key_tag"`
	Algorithm  types.String `tfsdk:"algorithm"`
	DigestType types.String `tfsdk:"digest_type"`
	Digest     types.String `tfsdk:"digest"`
}

func (r *DNSSECRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DNSSECRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DNSSECRecordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	record := porkbun.DNSSECRecord{
		KeyTag:     data.KeyTag.ValueString(),
		Alg:        data.Algorithm.ValueString(),
		DigestType: data.DigestType.ValueString(),
		Digest:     data.Digest.ValueString(),
	}

	err := r.client.CreateDNSSECRecord(ctx, data.Domain.ValueString(), record)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create DNSSEC record", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.Domain.ValueString(), data.KeyTag.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSSECRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DNSSECRecordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	records, err := r.client.GetDNSSECRecords(ctx, data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read DNSSEC records", err.Error())
		return
	}

	record, found := records[data.KeyTag.ValueString()]
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Algorithm = types.StringValue(record.Alg)
	data.DigestType = types.StringValue(record.DigestType)
	// Keep the state's digest when it only differs from the API's by hex
	// letter case: digest requires replacement on change, so adopting a
	// server-normalized casing into state would force a destroy/recreate
	// on every subsequent plan.
	if !strings.EqualFold(data.Digest.ValueString(), record.Digest) {
		data.Digest = types.StringValue(record.Digest)
	}
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.Domain.ValueString(), data.KeyTag.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is never invoked: every user-configurable attribute requires
// replacement because Porkbun's API has no DNSSEC update endpoint.
func (r *DNSSECRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DNSSECRecordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSSECRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DNSSECRecordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Porkbun keys DNSSEC records solely by key tag, so under
	// create_before_destroy the replacement record takes over this record's
	// identity before Delete runs. Only delete when the remote record still
	// matches this record's data.
	records, err := r.client.GetDNSSECRecords(ctx, data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read DNSSEC records", err.Error())
		return
	}

	record, found := records[data.KeyTag.ValueString()]
	if !found {
		return
	}
	if record.Alg != data.Algorithm.ValueString() ||
		record.DigestType != data.DigestType.ValueString() ||
		!strings.EqualFold(record.Digest, data.Digest.ValueString()) {
		return
	}

	err = r.client.DeleteDNSSECRecord(ctx, data.Domain.ValueString(), data.KeyTag.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete DNSSEC record", err.Error())
		return
	}
}

func (r *DNSSECRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: domain:key_tag. Got: %q", req.ID),
		)
		return
	}

	// Manually set attributes from the ID parts because the Read method relies on
	// "domain" and "key_tag" being present in the state to function correctly.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key_tag"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
