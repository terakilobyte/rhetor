package container

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/terakilobyte/rhetor/filesystem"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// ProvisionRequest contains information about the provision request
//
// Ports will be mapped in the range of 3000:5000 - 3999:5999
type ProvisionRequest struct {
	StudentID string `json:"studentID,omitempty"` // ID of the student
	DevPort   string `json:"devPort,omitempty"`   // The request port to map from host to container
	AppPort   string `json:"appPort,omitempty"`   // The request port to map from host to container
	Course    string `json:"course,omitempty"`    // The course request. This will be used to sync with S3
	FS        *filesystem.FSManager
	AWS       *session.Session
}

// DestroyRequest contains the id of the container to destroy
type DestroyRequest struct {
	ContainerID string `json:"containerID,omitempty"`
	StudentID   string `json:"studentID"`
	Port        int    `json:"port"`
	FS          *filesystem.FSManager
	AWS         *session.Session
}

// portOffset Used to offset the "application port" that will be bound.
//
// For example, if req.Port is 3599, the development port will be 3599, and the
// application port will be 5599
const portOffset int = 2000
const memoryLimit int64 = 2147483648 // 2GB (in bytes)

// ForwardedPort is the host port that is forwarded to the container
type ForwardedPort string

// Provision provisions a new ide container upon request
func Provision(req ProvisionRequest) (string, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", err
	}
	if err := req.FS.LoadStudentFilesDisk(req.AWS); err != nil {
		return "", err
	}
	imageName := "mflix-python:latest"
	t := 2 // timeout, 2 seconds
	tPtr := &t
	config := &container.Config{
		Image: imageName,
		ExposedPorts: nat.PortSet{
			"3000": struct{}{}, // expose container port 3000
			"5000": struct{}{}, // expose container port 5000
		},
		StopTimeout: tPtr,
	}
	if err != nil {
		return "", err
	}

	hostConfig := &container.HostConfig{
		// Bind the student's folder to the container
		Binds: []string{"/usr/local/share/rhetor/" + req.FS.StudentFSIdentifier + "/mflix-python:/home/project:cached"},
		PortBindings: nat.PortMap{
			// Bind container port 3000 to the assigned development port of the student on the host machine
			"3000": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: req.DevPort,
				},
			},
			// Bind container port 5000 to the assigned application port of the student on the host machine
			"5000": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: req.AppPort,
				},
			},
		},
		// Set a maximum memory usage of 2GB
		// Tuning required to see if 1GB or less is enough?
		Resources: container.Resources{
			Memory: memoryLimit,
		},
	}
	container, err := cli.ContainerCreate(ctx, config, hostConfig, nil, req.FS.StudentID)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	if err := cli.ContainerStart(ctx, container.ID, types.ContainerStartOptions{}); err != nil {
		fmt.Println(err)
		return "", err
	}
	return container.ID, nil
}

// Destroy kills and removes the specified container
func Destroy(req DestroyRequest) error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	if err := cli.ContainerStop(ctx, req.ContainerID, nil); err != nil {
		return err
	}
	if err := cli.ContainerRemove(ctx, req.ContainerID, types.ContainerRemoveOptions{}); err != nil {
		return err
	}
	if err := req.FS.SaveStudentFilesAWS(req.AWS); err != nil {
		return err
	}
	return nil
}
