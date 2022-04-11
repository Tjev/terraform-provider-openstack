package openstack

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
)

func TestComputeServerGroupV2CreateOpts(t *testing.T) {
	createOpts := ComputeServerGroupV2CreateOpts{
		servergroups.CreateOpts{
			Name:   "foo",
			Policy: "affinity",
		},
		map[string]string{
			"foo": "bar",
		},
	}

	expected := map[string]interface{}{
		"server_group": map[string]interface{}{
			"name":   "foo",
			"policy": "affinity",
			"foo":    "bar",
		},
	}

	actual, err := createOpts.ToServerGroupCreateMap()

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
