package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	compute "google.golang.org/api/compute/v1"
)

func init() {
	scopes := strings.Join([]string{
		compute.DevstorageFull_controlScope,
		compute.ComputeScope,
	}, " ")
	registerDemo("compute", scopes, computeMain)
}

func computeMain(client *http.Client, argv []string) {
	if len(argv) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: compute project_id instance_name (to start an instance)")
		return
	}

	service, _ := compute.New(client)
	projectId := argv[0]
	instanceName := argv[1]

	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectId
	imageURL := "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-7-wheezy-v20140606"
	zone := "us-central1-a"

	// Show the current images that are available.
	res, err := service.Images.List(projectId).Do()
	log.Printf("Got compute.Images.List, err: %#v, %v", res, err)

	instance := &compute.Instance{
		Name:        instanceName,
		Description: "compute sample instance",
		MachineType: prefix + "/zones/" + zone + "/machineTypes/n1-standard-1",
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskName:    "my-root-pd",
					SourceImage: imageURL,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			&compute.NetworkInterface{
				AccessConfigs: []*compute.AccessConfig{
					&compute.AccessConfig{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
				Network: prefix + "/global/networks/default",
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: "default",
				Scopes: []string{
					compute.DevstorageFull_controlScope,
					compute.ComputeScope,
				},
			},
		},
	}

	op, err := service.Instances.Insert(projectId, zone, instance).Do()
	log.Printf("Got compute.Operation, err: %#v, %v", op, err)
}
