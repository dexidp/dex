package testcontainers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/testcontainers/testcontainers-go/wait"
)

// Implement interfaces
var _ Container = (*DockerContainer)(nil)

// DockerContainer represents a container started using Docker
type DockerContainer struct {
	// Container ID from Docker
	ID         string
	WaitingFor wait.Strategy

	// Cache to retrieve container infromation without re-fetching them from dockerd
	raw               *types.ContainerJSON
	provider          *DockerProvider
	sessionID         uuid.UUID
	terminationSignal chan bool
	skipReaper        bool
}

func (c *DockerContainer) GetContainerID() string {
	return c.ID
}

// Endpoint gets proto://host:port string for the first exposed port
// Will returns just host:port if proto is ""
func (c *DockerContainer) Endpoint(ctx context.Context, proto string) (string, error) {
	ports, err := c.Ports(ctx)
	if err != nil {
		return "", err
	}

	// get first port
	var firstPort nat.Port
	for p := range ports {
		firstPort = p
		break
	}

	return c.PortEndpoint(ctx, firstPort, proto)
}

// PortEndpoint gets proto://host:port string for the given exposed port
// Will returns just host:port if proto is ""
func (c *DockerContainer) PortEndpoint(ctx context.Context, port nat.Port, proto string) (string, error) {
	host, err := c.Host(ctx)
	if err != nil {
		return "", err
	}

	outerPort, err := c.MappedPort(ctx, port)
	if err != nil {
		return "", err
	}

	protoFull := ""
	if proto != "" {
		protoFull = fmt.Sprintf("%s://", proto)
	}

	return fmt.Sprintf("%s%s:%s", protoFull, host, outerPort.Port()), nil
}

// Host gets host (ip or name) of the docker daemon where the container port is exposed
// Warning: this is based on your Docker host setting. Will fail if using an SSH tunnel
// You can use the "TC_HOST" env variable to set this yourself
func (c *DockerContainer) Host(ctx context.Context) (string, error) {
	host, err := c.provider.daemonHost()
	if err != nil {
		return "", err
	}
	return host, nil
}

// MappedPort gets externally mapped port for a container port
func (c *DockerContainer) MappedPort(ctx context.Context, port nat.Port) (nat.Port, error) {
	ports, err := c.Ports(ctx)
	if err != nil {
		return "", err
	}

	for k, p := range ports {
		if k.Port() != port.Port() {
			continue
		}
		if port.Proto() != "" && k.Proto() != port.Proto() {
			continue
		}
		return nat.NewPort(k.Proto(), p[0].HostPort)
	}

	return "", errors.New("port not found")
}

// Ports gets the exposed ports for the container.
func (c *DockerContainer) Ports(ctx context.Context) (nat.PortMap, error) {
	inspect, err := c.inspectContainer(ctx)
	if err != nil {
		return nil, err
	}
	return inspect.NetworkSettings.Ports, nil
}

// SessionID gets the current session id
func (c *DockerContainer) SessionID() string {
	return c.sessionID.String()
}

// Start will start an already created container
func (c *DockerContainer) Start(ctx context.Context) error {
	if err := c.provider.client.ContainerStart(ctx, c.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	// if a Wait Strategy has been specified, wait before returning
	if c.WaitingFor != nil {
		if err := c.WaitingFor.WaitUntilReady(ctx, c); err != nil {
			return err
		}
	}

	return nil
}

// Terminate is used to kill the container. It is usally triggered by as defer function.
func (c *DockerContainer) Terminate(ctx context.Context) error {
	err := c.provider.client.ContainerRemove(ctx, c.GetContainerID(), types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})

	if err == nil {
		c.sessionID = uuid.UUID{}
		c.raw = nil
	}

	return err
}

func (c *DockerContainer) inspectContainer(ctx context.Context) (*types.ContainerJSON, error) {
	if c.raw != nil {
		return c.raw, nil
	}
	inspect, err := c.provider.client.ContainerInspect(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.raw = &inspect

	return c.raw, nil
}

// Logs will fetch both STDOUT and STDERR from the current container. Returns a
// ReadCloser and leaves it up to the caller to extract what it wants.
func (c *DockerContainer) Logs(ctx context.Context) (io.ReadCloser, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}
	return c.provider.client.ContainerLogs(ctx, c.ID, options)
}

// Name gets the name of the container.
func (c *DockerContainer) Name(ctx context.Context) (string, error) {
	inspect, err := c.inspectContainer(ctx)
	if err != nil {
		return "", err
	}
	return inspect.Name, nil
}

// Networks gets the names of the networks the container is attached to.
func (c *DockerContainer) Networks(ctx context.Context) ([]string, error) {
	inspect, err := c.inspectContainer(ctx)
	if err != nil {
		return []string{}, err
	}

	networks := inspect.NetworkSettings.Networks

	n := []string{}

	for k := range networks {
		n = append(n, k)
	}

	return n, nil
}

// NetworkAliases gets the aliases of the container for the networks it is attached to.
func (c *DockerContainer) NetworkAliases(ctx context.Context) (map[string][]string, error) {
	inspect, err := c.inspectContainer(ctx)
	if err != nil {
		return map[string][]string{}, err
	}

	networks := inspect.NetworkSettings.Networks

	a := map[string][]string{}

	for k := range networks {
		a[k] = networks[k].Aliases
	}

	return a, nil
}

func (c *DockerContainer) Exec(ctx context.Context, cmd []string) (int, error) {
	cli := c.provider.client
	response, err := cli.ContainerExecCreate(ctx, c.ID, types.ExecConfig{
		Cmd:    cmd,
		Detach: false,
	})
	if err != nil {
		return 0, err
	}

	err = cli.ContainerExecStart(ctx, response.ID, types.ExecStartCheck{
		Detach: false,
	})
	if err != nil {
		return 0, err
	}

	var exitCode int
	for {
		execResp, err := cli.ContainerExecInspect(ctx, response.ID)
		if err != nil {
			return 0, err
		}

		if !execResp.Running {
			exitCode = execResp.ExitCode
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return exitCode, nil
}

// DockerNetwork represents a network started using Docker
type DockerNetwork struct {
	ID                string // Network ID from Docker
	Driver            string
	Name              string
	provider          *DockerProvider
	terminationSignal chan bool
}

// Remove is used to remove the network. It is usually triggered by as defer function.
func (n *DockerNetwork) Remove(_ context.Context) error {
	if n.terminationSignal != nil {
		n.terminationSignal <- true
	}
	return nil
}

// DockerProvider implements the ContainerProvider interface
type DockerProvider struct {
	client    *client.Client
	hostCache string
}

var _ ContainerProvider = (*DockerProvider)(nil)

// NewDockerProvider creates a Docker provider with the EnvClient
func NewDockerProvider() (*DockerProvider, error) {
	client, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	client.NegotiateAPIVersion(context.Background())
	p := &DockerProvider{
		client: client,
	}
	return p, nil
}

// BuildImage will build and image from context and Dockerfile, then return the tag
func (p *DockerProvider) BuildImage(ctx context.Context, img ImageBuildInfo) (string, error) {
	repo := uuid.NewV4()
	tag := uuid.NewV4()

	repoTag := fmt.Sprintf("%s:%s", repo, tag)

	buildContext, err := img.GetContext()
	if err != nil {
		return "", err
	}

	buildOptions := types.ImageBuildOptions{
		Dockerfile: img.GetDockerfile(),
		Context:    buildContext,
		Tags:       []string{repoTag},
	}

	resp, err := p.client.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		return "", err
	}

	// need to read the response from Docker, I think otherwise the image
	// might not finish building before continuing to execute here
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}

	resp.Body.Close()

	return repoTag, nil
}

// CreateContainer fulfills a request for a container without starting it
func (p *DockerProvider) CreateContainer(ctx context.Context, req ContainerRequest) (Container, error) {
	exposedPortSet, exposedPortMap, err := nat.ParsePortSpecs(req.ExposedPorts)
	if err != nil {
		return nil, err
	}

	env := []string{}
	for envKey, envVar := range req.Env {
		env = append(env, envKey+"="+envVar)
	}

	if req.Labels == nil {
		req.Labels = make(map[string]string)
	}

	sessionID := uuid.NewV4()

	var termSignal chan bool
	if !req.SkipReaper {
		r, err := NewReaper(ctx, sessionID.String(), p)
		if err != nil {
			return nil, errors.Wrap(err, "creating reaper failed")
		}
		termSignal, err = r.Connect()
		if err != nil {
			return nil, errors.Wrap(err, "connecting to reaper failed")
		}
		for k, v := range r.Labels() {
			if _, ok := req.Labels[k]; !ok {
				req.Labels[k] = v
			}
		}
	}

	if err = req.Validate(); err != nil {
		return nil, err
	}

	var tag string
	if req.ShouldBuildImage() {
		tag, err = p.BuildImage(ctx, &req)
		if err != nil {
			return nil, err
		}
	} else {
		tag = req.Image
		_, _, err = p.client.ImageInspectWithRaw(ctx, tag)
		if err != nil {
			if client.IsErrNotFound(err) {
				pullOpt := types.ImagePullOptions{}
				if req.RegistryCred != "" {
					pullOpt.RegistryAuth = req.RegistryCred
				}
				var pull io.ReadCloser
				err := backoff.Retry(func() error {
					var err error
					pull, err = p.client.ImagePull(ctx, tag, pullOpt)
					return err
				}, backoff.NewExponentialBackOff())
				if err != nil {
					return nil, err
				}
				defer pull.Close()

				// download of docker image finishes at EOF of the pull request
				_, err = ioutil.ReadAll(pull)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	dockerInput := &container.Config{
		Image:        tag,
		Env:          env,
		ExposedPorts: exposedPortSet,
		Labels:       req.Labels,
		Cmd:          req.Cmd,
	}

	// prepare mounts
	bindMounts := []mount.Mount{}
	for hostPath, innerPath := range req.BindMounts {
		bindMounts = append(bindMounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: hostPath,
			Target: innerPath,
		})
	}

	hostConfig := &container.HostConfig{
		PortBindings: exposedPortMap,
		Mounts:       bindMounts,
		AutoRemove:   true,
		Privileged:   req.Privileged,
	}

	endpointConfigs := map[string]*network.EndpointSettings{}
	for _, n := range req.Networks {
		nw, err := p.GetNetwork(ctx, NetworkRequest{
			Name: n,
		})
		if err == nil {
			endpointSetting := network.EndpointSettings{
				Aliases:   req.NetworkAliases[n],
				NetworkID: nw.ID,
			}
			endpointConfigs[n] = &endpointSetting
		}
	}
	networkingConfig := network.NetworkingConfig{
		EndpointsConfig: endpointConfigs,
	}

	resp, err := p.client.ContainerCreate(ctx, dockerInput, hostConfig, &networkingConfig, req.Name)
	if err != nil {
		return nil, err
	}

	c := &DockerContainer{
		ID:                resp.ID,
		WaitingFor:        req.WaitingFor,
		sessionID:         sessionID,
		provider:          p,
		terminationSignal: termSignal,
		skipReaper:        req.SkipReaper,
	}

	return c, nil
}

// RunContainer takes a RequestContainer as input and it runs a container via the docker sdk
func (p *DockerProvider) RunContainer(ctx context.Context, req ContainerRequest) (Container, error) {
	c, err := p.CreateContainer(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := c.Start(ctx); err != nil {
		return c, errors.Wrap(err, "could not start container")
	}

	return c, nil
}

// daemonHost gets the host or ip of the Docker daemon where ports are exposed on
// Warning: this is based on your Docker host setting. Will fail if using an SSH tunnel
// You can use the "TC_HOST" env variable to set this yourself
func (p *DockerProvider) daemonHost() (string, error) {
	if p.hostCache != "" {
		return p.hostCache, nil
	}

	host, exists := os.LookupEnv("TC_HOST")
	if exists {
		p.hostCache = host
		return p.hostCache, nil
	}

	// infer from Docker host
	url, err := url.Parse(p.client.DaemonHost())
	if err != nil {
		return "", err
	}

	switch url.Scheme {
	case "http", "https", "tcp":
		p.hostCache = url.Hostname()
	case "unix", "npipe":
		if inAContainer() {
			ip, err := getGatewayIp()
			if err != nil {
				return "", err
			}
			p.hostCache = ip
		} else {
			p.hostCache = "localhost"
		}
	default:
		return "", errors.New("Could not determine host through env or docker host")
	}

	return p.hostCache, nil
}

// CreateNetwork returns the object representing a new network identified by its name
func (p *DockerProvider) CreateNetwork(ctx context.Context, req NetworkRequest) (Network, error) {
	if req.Labels == nil {
		req.Labels = make(map[string]string)
	}

	nc := types.NetworkCreate{
		Driver:         req.Driver,
		CheckDuplicate: req.CheckDuplicate,
		Internal:       req.Internal,
		EnableIPv6:     req.EnableIPv6,
		Attachable:     req.Attachable,
		Labels:         req.Labels,
	}

	sessionID := uuid.NewV4()

	var termSignal chan bool
	if !req.SkipReaper {
		r, err := NewReaper(ctx, sessionID.String(), p)
		if err != nil {
			return nil, errors.Wrap(err, "creating network reaper failed")
		}
		termSignal, err = r.Connect()
		if err != nil {
			return nil, errors.Wrap(err, "connecting to network reaper failed")
		}
		for k, v := range r.Labels() {
			if _, ok := req.Labels[k]; !ok {
				req.Labels[k] = v
			}
		}
	}

	response, err := p.client.NetworkCreate(ctx, req.Name, nc)
	if err != nil {
		return &DockerNetwork{}, err
	}

	n := &DockerNetwork{
		ID:                response.ID,
		Driver:            req.Driver,
		Name:              req.Name,
		terminationSignal: termSignal,
	}

	return n, nil
}

// GetNetwork returns the object representing the network identified by its name
func (p *DockerProvider) GetNetwork(ctx context.Context, req NetworkRequest) (types.NetworkResource, error) {
	networkResource, err := p.client.NetworkInspect(ctx, req.Name, types.NetworkInspectOptions{
		Verbose: true,
	})
	if err != nil {
		return types.NetworkResource{}, err
	}

	return networkResource, err
}

func inAContainer() bool {
	// see https://github.com/testcontainers/testcontainers-java/blob/3ad8d80e2484864e554744a4800a81f6b7982168/core/src/main/java/org/testcontainers/dockerclient/DockerClientConfigUtils.java#L15
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

func getGatewayIp() (string, error) {
	// see https://github.com/testcontainers/testcontainers-java/blob/3ad8d80e2484864e554744a4800a81f6b7982168/core/src/main/java/org/testcontainers/dockerclient/DockerClientConfigUtils.java#L27
	cmd := exec.Command("sh", "-c", "ip route|awk '/default/ { print $3 }'")
	stdout, err := cmd.Output()
	if err != nil {
		return "", errors.New("Failed to detect docker host")
	}
	ip := strings.TrimSpace(string(stdout))
	if len(ip) == 0 {
		return "", errors.New("Failed to parse default gateway IP")
	}
	return string(ip), nil
}
