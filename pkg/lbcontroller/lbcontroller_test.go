/*
Copyright 2024 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lbcontroller

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResyncIPMap(t *testing.T) {

	cases := []struct {
		name         string
		ipList       []string
		clusterNodes []TestNode
		expectedMap  map[string]int
		expectedErr  bool
	}{
		{
			name:   "Nodes do not have annotation",
			ipList: []string{},
			clusterNodes: []TestNode{
				{
					Name: "node-1",
				},
			},
			expectedMap: map[string]int{},
		},
		{
			name:   "Nodes have IP annotation, IP not in the ipList",
			ipList: []string{"127.0.0.1"},
			clusterNodes: []TestNode{
				{
					Name:       "node-1",
					AssignedIP: "127.0.0.0",
				},
			},
			expectedMap: map[string]int{"127.0.0.1": 0},
		},
		{
			name:   "Nodes have IP annotation, IP in the ipList",
			ipList: []string{"127.0.0.1"},
			clusterNodes: []TestNode{
				{
					Name:       "node-1",
					AssignedIP: "127.0.0.1",
				},
			},
			expectedMap: map[string]int{"127.0.0.1": 1},
		},
		{
			name:   "Nodes have IP annotation, part of the IPs are in the ipList",
			ipList: []string{"127.0.0.1", "192.168.1.1"},
			clusterNodes: []TestNode{
				{
					Name:       "node-1",
					AssignedIP: "127.0.0.1",
				},
				{
					Name:       "node-2",
					AssignedIP: "127.0.0.0",
				},
				{
					Name: "node-3",
				},
			},
			expectedMap: map[string]int{"127.0.0.1": 1, "192.168.1.1": 0},
		},
	}
	for _, test := range cases {
		nodePool := NewNodePool(test.clusterNodes)
		lbController := NewFakeLBController(map[string]int{}, nodePool)
		ipMap, err := lbController.resyncIPMap(test.ipList)
		if gotExpected := gotExpectedError(test.name, test.expectedErr, err); gotExpected != nil {
			t.Fatal(gotExpected)
		}
		if diff := cmp.Diff(test.expectedMap, ipMap); diff != "" {
			t.Errorf("test %q failed: unexpected diff (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestAssignIPToNode(t *testing.T) {
	dummyVolID := "vol-1"
	cases := []struct {
		name         string
		ipMap        map[string]int
		clusterNodes []TestNode
		nodeName     string
		expectedIP   string
		expectedMap  map[string]int
		expectedErr  bool
		// TODO: validate the updated node object.
	}{
		{
			name: "Node already have IP assigned, IP found in ipMap",
			ipMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 1,
				"127.0.0.3": 1,
			},
			clusterNodes: []TestNode{
				{
					Name:       "node-1",
					AssignedIP: "127.0.0.2",
				},
				{
					Name:       "node-2",
					AssignedIP: "127.0.0.1",
				},
				{
					Name:       "node-3",
					AssignedIP: "127.0.0.3",
				},
			},
			nodeName:   "node-1",
			expectedIP: "127.0.0.2",
			expectedMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 1,
				"127.0.0.3": 1,
			},
		},
		{
			name: "Node already have IP assigned, IP not found in ipMap",
			ipMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 0,
				"127.0.0.3": 1,
			},
			clusterNodes: []TestNode{
				{
					Name:       "node-1",
					AssignedIP: "192.168.1.1",
				},
				{
					Name:       "node-2",
					AssignedIP: "127.0.0.1",
				},
				{
					Name:       "node-3",
					AssignedIP: "127.0.0.3",
				},
			},
			nodeName:   "node-1",
			expectedIP: "127.0.0.2",
			expectedMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 1,
				"127.0.0.3": 1,
			},
		},
		{
			name: "Node doesn't have IP assigned",
			ipMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 0,
				"127.0.0.3": 1,
			},
			clusterNodes: []TestNode{
				{
					Name: "node-1",
				},
				{
					Name:       "node-2",
					AssignedIP: "127.0.0.1",
				},
				{
					Name:       "node-3",
					AssignedIP: "127.0.0.3",
				},
			},
			nodeName:   "node-1",
			expectedIP: "127.0.0.2",
			expectedMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 1,
				"127.0.0.3": 1,
			},
		},
		// TODO: add a negative test case when node update fail.
	}
	for _, test := range cases {
		nodePool := NewNodePool(test.clusterNodes)
		lbController := NewFakeLBController(test.ipMap, nodePool)
		ctx := context.Background()
		ip, err := lbController.AssignIPToNode(ctx, test.nodeName, dummyVolID)
		if gotExpected := gotExpectedError(test.name, test.expectedErr, err); gotExpected != nil {
			t.Fatal(gotExpected)
		}
		if diff := cmp.Diff(test.expectedIP, ip); diff != "" {
			t.Errorf("test %q failed: unexpected diff (-want +got):\n%s", test.name, diff)
		}
		if diff := cmp.Diff(test.expectedMap, lbController.ipMap); diff != "" {
			t.Errorf("test %q failed: unexpected diff (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestRemoveIPFromNode(t *testing.T) {
	dummyVolID := "vol-1"
	cases := []struct {
		name         string
		ipMap        map[string]int
		clusterNodes []TestNode
		nodeName     string
		expectedMap  map[string]int
		expectedErr  bool
	}{
		{
			name: "node not found, removeIPFromNode skipped",
			ipMap: map[string]int{
				"127.0.0.1": 1,
			},
			nodeName: "node-1",
			expectedMap: map[string]int{
				"127.0.0.1": 1,
			},
		},
		{
			name: "Node does not have annotation",
			ipMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 0,
				"127.0.0.3": 1,
			},
			clusterNodes: []TestNode{
				{
					Name: "node-1",
				},
				{
					Name:       "node-2",
					AssignedIP: "127.0.0.1",
				},
				{
					Name:       "node-3",
					AssignedIP: "127.0.0.3",
				},
			},
			nodeName: "node-1",
			expectedMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 0,
				"127.0.0.3": 1,
			},
		},
		{
			name: "Node has IP annotation, but IP not found in ipMap",
			ipMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 0,
				"127.0.0.3": 1,
			},
			clusterNodes: []TestNode{
				{
					Name:       "node-1",
					AssignedIP: "192.168.1.1",
				},
				{
					Name:       "node-2",
					AssignedIP: "127.0.0.1",
				},
				{
					Name:       "node-3",
					AssignedIP: "127.0.0.3",
				},
			},
			nodeName: "node-1",
			expectedMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 0,
				"127.0.0.3": 1,
			},
		},
		{
			name: "Node has IP annotation, IP exist in ipMap",
			ipMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 1,
				"127.0.0.3": 1,
			},
			clusterNodes: []TestNode{
				{
					Name:       "node-1",
					AssignedIP: "127.0.0.2",
				},
				{
					Name:       "node-2",
					AssignedIP: "127.0.0.1",
				},
				{
					Name:       "node-3",
					AssignedIP: "127.0.0.3",
				},
			},
			nodeName: "node-1",
			expectedMap: map[string]int{
				"127.0.0.1": 1,
				"127.0.0.2": 0,
				"127.0.0.3": 1,
			},
		},
	}
	for _, test := range cases {
		nodePool := NewNodePool(test.clusterNodes)
		lbController := NewFakeLBController(test.ipMap, nodePool)
		ctx := context.Background()
		err := lbController.RemoveIPFromNode(ctx, test.nodeName, dummyVolID)
		if gotExpected := gotExpectedError(test.name, test.expectedErr, err); gotExpected != nil {
			t.Fatal(gotExpected)
		}
		if diff := cmp.Diff(test.expectedMap, lbController.ipMap); diff != "" {
			t.Errorf("test %q failed: unexpected diff (-want +got):\n%s", test.name, diff)
		}
	}
}

func gotExpectedError(testFunc string, wantErr bool, err error) error {
	if err != nil && !wantErr {
		return fmt.Errorf("%s got error %v, want nil", testFunc, err)
	}
	if err == nil && wantErr {
		return fmt.Errorf("%s got nil, want error", testFunc)
	}
	return nil
}
