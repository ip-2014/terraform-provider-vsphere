package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi/vim25/types"
)

// schemaHostPortGroupSpec returns schema items for resources that
// need to work with HostPortGroupSpec, such as port groups.
func schemaHostPortGroupSpec() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		// HostPortGroupSpec
		"name": &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "The name of the port group.",
			ForceNew:    true,
		},
		"vlan_id": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "The VLAN ID/trunk mode for this port group. An ID of 0 denotes no tagging, an ID of 1-4094 tags with the specific ID, and an ID of 4095 enables trunk mode, allowing the guest to manage its own tagging.",
			Default:     0,
		},
		"virtual_switch_name": &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "The name of the virtual switch to bind this port group to.",
			ForceNew:    true,
		},
	}
	mergeSchema(s, schemaHostNetworkPolicy())
	return s
}

// expandHostPortGroupSpec reads certain ResourceData keys and returns a
// HostPortGroupSpec.
func expandHostPortGroupSpec(d *schema.ResourceData) *types.HostPortGroupSpec {
	obj := &types.HostPortGroupSpec{
		Name:        d.Get("name").(string),
		VlanId:      int32(d.Get("vlan_id").(int)),
		VswitchName: d.Get("virtual_switch_name").(string),
		Policy:      *expandHostNetworkPolicy(d),
	}
	return obj
}

// flattenHostPortGroupSpec reads various fields from a HostPortGroupSpec into
// the passed in ResourceData.
func flattenHostPortGroupSpec(d *schema.ResourceData, obj *types.HostPortGroupSpec) error {
	d.Set("vlan_id", obj.VlanId)
	if err := flattenHostNetworkPolicy(d, &obj.Policy); err != nil {
		return err
	}
	return nil
}

// calculateComputedPolicy is a utility function to compute a map of state
// attributes for the port group's effective policy. It uses a bit of a
// roundabout way to set the attributes, but allows us to utilize our
// functional deep reading helpers to perform this task, versus having to
// re-write code.
//
// This function relies a bit on some of the lower-level utility functionality
// of helper/schema, so it may need to change in the future.
func calculateComputedPolicy(policy types.HostNetworkPolicy) (map[string]string, error) {
	cpr := &schema.Resource{Schema: schemaHostNetworkPolicy()}
	cpd := cpr.Data(&terraform.InstanceState{})
	cpd.SetId("effectivepolicy")
	if err := flattenHostNetworkPolicy(cpd, &policy); err != nil {
		return nil, fmt.Errorf("error setting effective policy data: %s", err)
	}
	cpm := cpd.State().Attributes
	delete(cpm, "id")
	return cpm, nil
}

// calculatePorts is a utility function that returns a set of port data.
func calculatePorts(ports []types.HostPortGroupPort) *schema.Set {
	s := make([]interface{}, 0)
	for _, port := range ports {
		m := make(map[string]interface{})
		m["key"] = port.Key
		m["mac_addresses"] = sliceStringsToInterfaces(port.Mac)
		m["type"] = port.Type
		s = append(s, m)
	}
	return schema.NewSet(schema.HashResource(portGroupPortSchema()), s)
}

// portGroupPortSchema returns a sub-schema for a port group's connected ports.
func portGroupPortSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"key": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The linkable identifier for this port entry.",
				Computed:    true,
			},
			"mac_addresses": &schema.Schema{
				Type:        schema.TypeList,
				Description: "The MAC addresses of the network service of the virtual machine connected on this port.",
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Type type of the entity connected on this port. Possible values are host (VMKkernel), systemManagement (service console), virtualMachine, or unknown.",
				Computed:    true,
			},
		},
	}
}
