// +build !integration,!e2e

// Copyright 2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package config

import (
	"net"
	"testing"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/stretchr/testify/assert"
)

type config struct {
	netConfig string
	pcArgs    string
}

var (
	validConfigs = []config{
		config{ // All required fields in netconfig.
			netConfig: `{"trunkName":"eth0", "branchVlanID":"100", "branchMACAddress":"01:23:45:67:89:ab", "branchIPAddress":"10.11.12.13/16", "uid":"42", "gid":"42"}`,
			pcArgs:    "",
		},
		config{ // All required network fields in netconfig and branch fields in per-container args.
			netConfig: `{"trunkName":"eth1", "uid":"42", "gid":"42"}`,
			pcArgs:    "BranchVlanID=10;BranchMACAddress=10:20:30:40:50:60;BranchIPAddress=192.168.1.2/16",
		},
		config{ // TrunkMACAddress instead of TrunkName.
			netConfig: `{"trunkMACAddress":"42:42:42:42:42:42", "branchVlanID":"100", "branchMACAddress":"01:23:45:67:89:ab", "branchIPAddress":"10.11.12.13/14", "uid":"42", "gid":"42"}`,
			pcArgs:    "",
		},
		config{ // With optional fields.
			netConfig: `{"trunkMACAddress":"42:42:42:42:42:42", "blockInstanceMetadata":true, "interfaceType":"tap", "uid":"42", "gid":"42"}`,
			pcArgs:    "BranchVlanID=10;BranchMACAddress=10:20:30:40:50:60;BranchIPAddress=192.168.1.2/24;BranchGatewayIPAddress=192.168.1.1",
		},
		config{ // VLAN interface with no TAP UID or GID.
			netConfig: `{"trunkName":"eth1", "interfaceType": "vlan"}`,
			pcArgs:    "BranchVlanID=10;BranchMACAddress=10:20:30:40:50:60;BranchIPAddress=192.168.1.2/16",
		},
	}

	invalidConfigs = []config{
		config{ // invalid branch IP address.
			netConfig: `{"trunkName":"eth1", "uid":"42", "gid":"42"}`,
			pcArgs:    "BranchVlanID=100;BranchMACAddress=10:20:30:40:50:60;BranchIPAddress=192.168.1/16",
		},
		config{ // missing branch VLAN ID.
			netConfig: `{"trunkName":"eth1", "uid":"42", "gid":"42"}`,
			pcArgs:    "BranchMACAddress=10:20:30:40:50:60;BranchIPAddress=192.168.1.2/16",
		},
		config{ // missing TAP UID and GID.
			netConfig: `{"trunkName":"eth1", "branchVlanID":"100", "interfaceType":"tap"}`,
			pcArgs:    "BranchMACAddress=10:20:30:40:50:60;BranchIPAddress=192.168.1.2/16",
		},
	}
)

// TestValidConfigs tests that valid configs succeed.
func TestValidConfigs(t *testing.T) {
	for _, config := range validConfigs {
		args := &skel.CmdArgs{
			StdinData: []byte(config.netConfig),
			Args:      config.pcArgs,
		}
		_, err := New(args)
		assert.NoError(t, err)
	}
}

// TestInvalidConfigs tests that invalid configs fail.
func TestInvalidConfigs(t *testing.T) {
	for _, config := range invalidConfigs {
		args := &skel.CmdArgs{
			StdinData: []byte(config.netConfig),
			Args:      config.pcArgs,
		}
		_, err := New(args)
		assert.Error(t, err)
	}
}

// TestPerContainerArgsOverrideNetConfig tests that per-container args override per-network args.
func TestPerContainerArgsOverrideNetConfig(t *testing.T) {
	c := config{
		netConfig: `{"trunkName":"eth0", "branchVlanID":"100", "branchMACAddress":"01:23:45:67:89:ab", "branchIPAddress":"10.11.12.13/14", "uid":"42", "gid":"42"}`,
		pcArgs:    "BranchVlanID=42;BranchMACAddress=44:44:44:55:55:55;BranchIPAddress=192.168.1.2/16",
	}

	args := &skel.CmdArgs{
		StdinData: []byte(c.netConfig),
		Args:      c.pcArgs,
	}
	nc, err := New(args)
	assert.NoError(t, err)

	assert.Equal(t, 42, nc.BranchVlanID, "invalid vlanid")
	assert.Equal(t, "44:44:44:55:55:55", nc.BranchMACAddress.String(), "invalid macaddress")
	assert.Equal(t, "192.168.1.2/16", nc.BranchIPAddress.String(), "invalid ipaddress")
}

func TestGetGatewayIPAddress(t *testing.T) {
	_, ipv4Net, err := net.ParseCIDR("172.31.16.3/20")
	assert.NoError(t, err)

	expectedGatewayIPAddress := net.ParseIP("172.31.16.2")

	outputGatewayIPAddress, err := getGatewayIPAddress(ipv4Net, "172.31.16.2")
	assert.NoError(t, err)
	assert.Equal(t, expectedGatewayIPAddress, outputGatewayIPAddress)
}

func TestGetGatewayIPAddressFromSubnet(t *testing.T) {
	_, ipv4Net, err := net.ParseCIDR("172.31.16.3/20")
	assert.NoError(t, err)

	expectedGatewayIPAddress := net.ParseIP("172.31.16.1")

	outputGatewayIPAddress, err := getGatewayIPAddress(ipv4Net, "")
	assert.NoError(t, err)
	assert.Equal(t, expectedGatewayIPAddress, outputGatewayIPAddress)
}
