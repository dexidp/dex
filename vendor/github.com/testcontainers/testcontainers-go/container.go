package testcontainers

import (
	"context"
	"io"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DeprecatedContainer shows methods that were supported before, but are now deprecated
// Deprecated: Use Container
type DeprecatedContainer interface {
	GetHostEndpoint(ctx context.Context, port string) (string, string, error)
	GetIPAddress(ctx context.Context) (string, error)
	LivenessCheckPorts(ctx context.Context) (nat.PortSet, error)
	Terminate(ctx context.Context) error
}

// ContainerProvider allows the creation of containers on an arbitrary system
type ContainerProvider interface {
	CreateContainer(context.Context, ContainerRequest) (Container, error) // create a container without starting it
	RunContainer(context.Context, ContainerRequest) (Container, error)    // create a container and start it
}

// Container allows getting info about and controlling a single container instance
type Container interface {
	GetContainerID() string                                         // get the container id from the provider
	Endpoint(context.Context, string) (string, error)               // get proto://ip:port string for the first exposed port
	PortEndpoint(context.Context, nat.Port, string) (string, error) // get proto://ip:port string for the given exposed port
	Host(context.Context) (string, error)                           // get host where the container port is exposed
	MappedPort(context.Context, nat.Port) (nat.Port, error)         // get externally mapped port for a container port
	Ports(context.Context) (nat.PortMap, error)                     // get all exposed ports
	SessionID() string                                              // get session id
	Start(context.Context) error                                    // start the container
	Terminate(context.Context) error                                // terminate the container
	Logs(context.Context) (io.ReadCloser, error)                    // Get logs of the container
	Name(context.Context) (string, error)                           // get container name
	Networks(context.Context) ([]string, error)                     // get container networks
	NetworkAliases(context.Context) (map[string][]string, error)    // get container network aliases for a network
	Exec(ctx context.Context, cmd []string) (int, error)
}

// ImageBuildInfo defines what is needed to build an image
type ImageBuildInfo interface {
	GetContext() (io.Reader, error) // the path to the build context
	GetDockerfile() string          // the relative path to the Dockerfile, including the fileitself
	ShouldBuildImage() bool         // return true if the image needs to be built
}

// FromDockerfile represents the parameters needed to build an image from a Dockerfile
// rather than using a pre-built one
type FromDockerfile struct {
	Context        string    // the path to the context of of the docker build
	ContextArchive io.Reader // the tar archive file to send to docker that contains the build context
	Dockerfile     string    // the path from the context to the Dockerfile for the image, defaults to "Dockerfile"
}

// ContainerRequest represents the parameters used to get a running container
type ContainerRequest struct {
	FromDockerfile
	Image          string
	Env            map[string]string
	ExposedPorts   []string // allow specifying protocol info
	Cmd            []string
	Labels         map[string]string
	BindMounts     map[string]string
	RegistryCred   string
	WaitingFor     wait.Strategy
	Name           string              // for specifying container name
	Privileged     bool                // for starting privileged container
	Networks       []string            // for specifying network names
	NetworkAliases map[string][]string // for specifying network aliases
	SkipReaper     bool                // indicates whether we skip setting up a reaper for this
}

// ProviderType is an enum for the possible providers
type ProviderType int

// possible provider types
const (
	ProviderDocker ProviderType = iota // Docker is default = 0
)

// GetProvider provides the provider implementation for a certain type
func (t ProviderType) GetProvider() (GenericProvider, error) {
	switch t {
	case ProviderDocker:
		provider, err := NewDockerProvider()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create Docker provider")
		}
		return provider, nil
	}
	return nil, errors.New("unknown provider")
}

// Validate ensures that the ContainerRequest does not have invalid paramters configured to it
// ex. make sure you are not specifying both an image as well as a context
func (c *ContainerRequest) Validate() error {

	validationMethods := []func() error{
		c.validateContextAndImage,
		c.validateContexOrImageIsSpecified,
	}

	var err error
	for _, validationMethod := range validationMethods {
		err = validationMethod()
		if err != nil {
			return err
		}
	}

	return nil
}

// GetContext retrieve the build context for the request
func (c *ContainerRequest) GetContext() (io.Reader, error) {
	if c.ContextArchive != nil {
		return c.ContextArchive, nil
	}

	buildContext, err := archive.TarWithOptions(c.Context, &archive.TarOptions{})
	if err != nil {
		return nil, err
	}

	return buildContext, nil
}

// GetDockerfile returns the Dockerfile from the ContainerRequest, defaults to "Dockerfile"
func (c *ContainerRequest) GetDockerfile() string {
	f := c.FromDockerfile.Dockerfile
	if f == "" {
		return "Dockerfile"
	}

	return f
}

func (c *ContainerRequest) ShouldBuildImage() bool {
	return c.FromDockerfile.Context != "" || c.FromDockerfile.ContextArchive != nil
}

func (c *ContainerRequest) validateContextAndImage() error {
	if c.FromDockerfile.Context != "" && c.Image != "" {
		return errors.New("you cannot specify both an Image and Context in a ContainerRequest")
	}

	return nil
}

func (c *ContainerRequest) validateContexOrImageIsSpecified() error {
	if c.FromDockerfile.Context == "" && c.FromDockerfile.ContextArchive == nil && c.Image == "" {
		return errors.New("you must specify either a build context or an image")
	}

	return nil
}
