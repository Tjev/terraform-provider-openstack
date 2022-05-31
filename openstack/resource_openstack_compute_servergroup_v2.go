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

			"policies": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"policy": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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
							Optional:     true,
							ForceNew:     true,
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

	name := d.Get("name").(string)

	rawPolicies := d.Get("policies").([]interface{})
	policies := expandComputeServerGroupV2Policies(computeClient, rawPolicies)

	policy := d.Get("policy").(string)
	rules_v, rules_set := d.GetOk("rules")

	var createOpts ComputeServerGroupV2CreateOpts

	// "policies" is replaced with "policy" and optional "rules" since microversion 2.64
	if len(policies) > 0 {
		if policy != "" {
			return diag.Errorf("Cannot create with both \"policies\" and \"policy\" field specified")
		}
		if rules_set {
			return diag.Errorf("Cannot use \"policies\" field with \"rules\"" +
				" - omit the \"rules\" or use \"policy\" instead")
		}
		createOpts = ComputeServerGroupV2CreateOpts{
			servergroups.CreateOpts{
				Name:     name,
				Policies: policies,
			},
			MapValueSpecs(d),
		}
	} else {
		computeClient.Microversion = "2.64"

		if policy == "anti-affinity" && rules_set {
			rules := rules_v.([]map[string]interface{})

			var max_server_per_host int
			if v, ok := rules[0]["max_server_per_host"]; ok {
				max_server_per_host = v.(int)
			}

			createOpts = ComputeServerGroupV2CreateOpts{
				servergroups.CreateOpts{
					Name:   name,
					Policy: policy,
					Rules: &servergroups.Rules{
						MaxServerPerHost: max_server_per_host,
					},
				},
				MapValueSpecs(d),
			}
		} else {
			createOpts = ComputeServerGroupV2CreateOpts{
				servergroups.CreateOpts{
					Name:   name,
					Policy: policy,
				},
				MapValueSpecs(d),
			}
		}
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

	sg, err := servergroups.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return diag.FromErr(CheckDeleted(d, err, "Error retrieving openstack_compute_servergroup_v2"))
	}

	log.Printf("[DEBUG] Retrieved openstack_compute_servergroup_v2 %s: %#v", d.Id(), sg)

	d.Set("name", sg.Name)

	if len(sg.Policies) > 0 {
		d.Set("policy", sg.Policies)
	}

	d.Set("members", sg.Members)

	d.Set("region", GetRegion(d, config))

	if sg.Policy != nil {
		d.Set("policy", sg.Policy)
	}

	if sg.Rules != nil {
		rules := make(map[string]interface{})
		rules["max_server_per_host"] = sg.Rules.MaxServerPerHost
		rules_l := []map[string]interface{}{rules}
		d.Set("rules", rules_l)
	}

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
