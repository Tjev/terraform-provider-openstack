package openstack

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
)

func resourceComputeServerGroupV2() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceComputeServerGroupV2Create,
		ReadContext:   resourceComputeServerGroupV2Read,
		Update:        nil,
		DeleteContext: resourceComputeServerGroupV2Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"policy": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"rules": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_server_per_host": {
							Type:         schema.TypeInt,
							ForceNew:     true,
							Optional:     true,
							Default:      1,
							ValidateFunc: validation.IntAtLeast(1),
						},
					},
				},
			},

			"members": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"value_specs": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceComputeServerGroupV2Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*Config)
	computeClient, err := config.ComputeV2Client(GetRegion(d, config))
	if err != nil {
		return diag.Errorf("Error creating OpenStack compute client: %s", err)
	}
	computeClient.Microversion = computeV2ServerGroupMinMicroversion

	name := d.Get("name").(string)

	// Add the port binding parameters if specified.
	var rules servergroups.Rules
	if r, ok := d.GetOk("rules"); ok {
		rV := (r.([]interface{}))[0].(map[string]interface{})
		rules = servergroups.Rules{
			MaxServerPerHost: rV["max_server_per_host"].(int),
		}
	}

	createOpts := ComputeServerGroupV2CreateOpts{
		servergroups.CreateOpts{
			Name:   name,
			Policy: d.Get("policy").(string),
		},
		MapValueSpecs(d),
	}

	if rules != (servergroups.Rules{}) {
		createOpts.Rules = &rules
	}

	log.Printf("[DEBUG] openstack_compute_servergroup_v2 create options: %#v", createOpts)
	newSG, err := servergroups.Create(computeClient, createOpts).Extract()
	if err != nil {
		return diag.Errorf("Error creating openstack_compute_servergroup_v2 %s: %s", name, err)
	}

	d.SetId(newSG.ID)

	return resourceComputeServerGroupV2Read(ctx, d, meta)
}

func resourceComputeServerGroupV2Read(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*Config)
	computeClient, err := config.ComputeV2Client(GetRegion(d, config))
	if err != nil {
		return diag.Errorf("Error creating OpenStack compute client: %s", err)
	}
	computeClient.Microversion = computeV2ServerGroupMinMicroversion

	sg, err := servergroups.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return diag.FromErr(CheckDeleted(d, err, "Error retrieving openstack_compute_servergroup_v2"))
	}

	log.Printf("[DEBUG] Retrieved openstack_compute_servergroup_v2 %s: %#v", d.Id(), sg)

	d.Set("name", sg.Name)
	d.Set("policy", sg.Policy)
	d.Set("rules", sg.Rules)
	d.Set("members", sg.Members)

	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceComputeServerGroupV2Delete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*Config)
	computeClient, err := config.ComputeV2Client(GetRegion(d, config))
	if err != nil {
		return diag.Errorf("Error creating OpenStack compute client: %s", err)
	}

	if err := servergroups.Delete(computeClient, d.Id()).ExtractErr(); err != nil {
		return diag.FromErr(CheckDeleted(d, err, "Error deleting openstack_compute_servergroup_v2"))
	}

	return nil
}
