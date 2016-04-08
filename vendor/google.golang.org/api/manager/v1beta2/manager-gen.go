// Package manager provides access to the Deployment Manager API.
//
// See https://developers.google.com/deployment-manager/
//
// Usage example:
//
//   import "google.golang.org/api/manager/v1beta2"
//   ...
//   managerService, err := manager.New(oauthHttpClient)
package manager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/googleapi"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Always reference these packages, just in case the auto-generated code
// below doesn't.
var _ = bytes.NewBuffer
var _ = strconv.Itoa
var _ = fmt.Sprintf
var _ = json.NewDecoder
var _ = io.Copy
var _ = url.Parse
var _ = googleapi.Version
var _ = errors.New
var _ = strings.Replace

const apiId = "manager:v1beta2"
const apiName = "manager"
const apiVersion = "v1beta2"
const basePath = "https://www.googleapis.com/manager/v1beta2/projects/"

// OAuth2 scopes used by this API.
const (
	// View and manage your applications deployed on Google App Engine
	AppengineAdminScope = "https://www.googleapis.com/auth/appengine.admin"

	// View and manage your data across Google Cloud Platform services
	CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

	// View and manage your Google Compute Engine resources
	ComputeScope = "https://www.googleapis.com/auth/compute"

	// Manage your data in Google Cloud Storage
	DevstorageRead_writeScope = "https://www.googleapis.com/auth/devstorage.read_write"

	// View and manage your Google Cloud Platform management resources and
	// deployment status information
	NdevCloudmanScope = "https://www.googleapis.com/auth/ndev.cloudman"

	// View your Google Cloud Platform management resources and deployment
	// status information
	NdevCloudmanReadonlyScope = "https://www.googleapis.com/auth/ndev.cloudman.readonly"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Deployments = NewDeploymentsService(s)
	s.Templates = NewTemplatesService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Deployments *DeploymentsService

	Templates *TemplatesService
}

func NewDeploymentsService(s *Service) *DeploymentsService {
	rs := &DeploymentsService{s: s}
	return rs
}

type DeploymentsService struct {
	s *Service
}

func NewTemplatesService(s *Service) *TemplatesService {
	rs := &TemplatesService{s: s}
	return rs
}

type TemplatesService struct {
	s *Service
}

type AccessConfig struct {
	// Name: Name of this access configuration.
	Name string `json:"name,omitempty"`

	// NatIp: An external IP address associated with this instance.
	NatIp string `json:"natIp,omitempty"`

	// Type: Type of this access configuration file. (Currently only
	// ONE_TO_ONE_NAT is legal.)
	Type string `json:"type,omitempty"`
}

type Action struct {
	// Commands: A list of commands to run sequentially for this action.
	Commands []string `json:"commands,omitempty"`

	// TimeoutMs: The timeout in milliseconds for this action to run.
	TimeoutMs int64 `json:"timeoutMs,omitempty"`
}

type AllowedRule struct {
	// IPProtocol: ?tcp?, ?udp? or ?icmp?
	IPProtocol string `json:"IPProtocol,omitempty"`

	// Ports: List of ports or port ranges (Example inputs include: ["22"],
	// [?33?, "12345-12349"].
	Ports []string `json:"ports,omitempty"`
}

type AutoscalingModule struct {
	CoolDownPeriodSec int64 `json:"coolDownPeriodSec,omitempty"`

	Description string `json:"description,omitempty"`

	MaxNumReplicas int64 `json:"maxNumReplicas,omitempty"`

	MinNumReplicas int64 `json:"minNumReplicas,omitempty"`

	SignalType string `json:"signalType,omitempty"`

	TargetModule string `json:"targetModule,omitempty"`

	// TargetUtilization: target_utilization should be in range [0,1].
	TargetUtilization float64 `json:"targetUtilization,omitempty"`
}

type AutoscalingModuleStatus struct {
	// AutoscalingConfigUrl: [Output Only] The URL of the corresponding
	// Autoscaling configuration.
	AutoscalingConfigUrl string `json:"autoscalingConfigUrl,omitempty"`
}

type DeployState struct {
	// Details: [Output Only] Human readable details about the current
	// state.
	Details string `json:"details,omitempty"`

	// Status: [Output Only] The status of the deployment. Possible values
	// include:
	// - UNKNOWN
	// - DEPLOYING
	// - DEPLOYED
	// - DEPLOYMENT_FAILED
	// -
	// DELETING
	// - DELETED
	// - DELETE_FAILED
	Status string `json:"status,omitempty"`
}

type Deployment struct {
	// CreationDate: [Output Only] The time when this deployment was
	// created.
	CreationDate string `json:"creationDate,omitempty"`

	// Description: A user-supplied description of this Deployment.
	Description string `json:"description,omitempty"`

	// Modules: [Output Only] List of status for the modules in this
	// deployment.
	Modules map[string]ModuleStatus `json:"modules,omitempty"`

	// Name: Name of this deployment. The name must conform to the following
	// regular expression: [a-zA-Z0-9-_]{1,64}
	Name string `json:"name,omitempty"`

	// Overrides: The set of parameter overrides to apply to the
	// corresponding Template before deploying.
	Overrides []*ParamOverride `json:"overrides,omitempty"`

	// State: [Output Only] Current status of this deployment.
	State *DeployState `json:"state,omitempty"`

	// TemplateName: The name of the Template on which this deployment is
	// based.
	TemplateName string `json:"templateName,omitempty"`
}

type DeploymentsListResponse struct {
	NextPageToken string `json:"nextPageToken,omitempty"`

	Resources []*Deployment `json:"resources,omitempty"`
}

type DiskAttachment struct {
	// DeviceName: The device name of this disk.
	DeviceName string `json:"deviceName,omitempty"`

	// Index: A zero-based index to assign to this disk, where 0 is reserved
	// for the boot disk. If not specified, this is assigned by the server.
	Index int64 `json:"index,omitempty"`
}

type EnvVariable struct {
	// Hidden: Whether this variable is hidden or visible.
	Hidden bool `json:"hidden,omitempty"`

	// Value: Value of the environment variable.
	Value string `json:"value,omitempty"`
}

type ExistingDisk struct {
	// Attachment: Optional. How the disk will be attached to the Replica.
	Attachment *DiskAttachment `json:"attachment,omitempty"`

	// Source: The fully-qualified URL of the Persistent Disk resource. It
	// must be in the same zone as the Pool.
	Source string `json:"source,omitempty"`
}

type FirewallModule struct {
	// Allowed: The allowed ports or port ranges.
	Allowed []*AllowedRule `json:"allowed,omitempty"`

	// Description: The description of the firewall (optional)
	Description string `json:"description,omitempty"`

	// Network: The NetworkModule to which this firewall should apply. If
	// not specified, or if specified as 'default', this firewall will be
	// applied to the 'default' network.
	Network string `json:"network,omitempty"`

	// SourceRanges: Source IP ranges to apply this firewall to, see the GCE
	// Spec for details on syntax
	SourceRanges []string `json:"sourceRanges,omitempty"`

	// SourceTags: Source Tags to apply this firewall to, see the GCE Spec
	// for details on syntax
	SourceTags []string `json:"sourceTags,omitempty"`

	// TargetTags: Target Tags to apply this firewall to, see the GCE Spec
	// for details on syntax
	TargetTags []string `json:"targetTags,omitempty"`
}

type FirewallModuleStatus struct {
	// FirewallUrl: [Output Only] The URL of the corresponding Firewall
	// resource.
	FirewallUrl string `json:"firewallUrl,omitempty"`
}

type HealthCheckModule struct {
	CheckIntervalSec int64 `json:"checkIntervalSec,omitempty"`

	Description string `json:"description,omitempty"`

	HealthyThreshold int64 `json:"healthyThreshold,omitempty"`

	Host string `json:"host,omitempty"`

	Path string `json:"path,omitempty"`

	Port int64 `json:"port,omitempty"`

	TimeoutSec int64 `json:"timeoutSec,omitempty"`

	UnhealthyThreshold int64 `json:"unhealthyThreshold,omitempty"`
}

type HealthCheckModuleStatus struct {
	// HealthCheckUrl: [Output Only] The HealthCheck URL.
	HealthCheckUrl string `json:"healthCheckUrl,omitempty"`
}

type LbModule struct {
	Description string `json:"description,omitempty"`

	HealthChecks []string `json:"healthChecks,omitempty"`

	IpAddress string `json:"ipAddress,omitempty"`

	IpProtocol string `json:"ipProtocol,omitempty"`

	PortRange string `json:"portRange,omitempty"`

	SessionAffinity string `json:"sessionAffinity,omitempty"`

	TargetModules []string `json:"targetModules,omitempty"`
}

type LbModuleStatus struct {
	// ForwardingRuleUrl: [Output Only] The URL of the corresponding
	// ForwardingRule in GCE.
	ForwardingRuleUrl string `json:"forwardingRuleUrl,omitempty"`

	// TargetPoolUrl: [Output Only] The URL of the corresponding TargetPool
	// resource in GCE.
	TargetPoolUrl string `json:"targetPoolUrl,omitempty"`
}

type Metadata struct {
	// FingerPrint: The fingerprint of the metadata.
	FingerPrint string `json:"fingerPrint,omitempty"`

	// Items: A list of metadata items.
	Items []*MetadataItem `json:"items,omitempty"`
}

type MetadataItem struct {
	// Key: A metadata key.
	Key string `json:"key,omitempty"`

	// Value: A metadata value.
	Value string `json:"value,omitempty"`
}

type Module struct {
	AutoscalingModule *AutoscalingModule `json:"autoscalingModule,omitempty"`

	FirewallModule *FirewallModule `json:"firewallModule,omitempty"`

	HealthCheckModule *HealthCheckModule `json:"healthCheckModule,omitempty"`

	LbModule *LbModule `json:"lbModule,omitempty"`

	NetworkModule *NetworkModule `json:"networkModule,omitempty"`

	ReplicaPoolModule *ReplicaPoolModule `json:"replicaPoolModule,omitempty"`

	// Type: The type of this module. Valid values ("AUTOSCALING",
	// "FIREWALL", "HEALTH_CHECK", "LOAD_BALANCING", "NETWORK",
	// "REPLICA_POOL")
	Type string `json:"type,omitempty"`
}

type ModuleStatus struct {
	// AutoscalingModuleStatus: [Output Only] The status of the
	// AutoscalingModule, set for type AUTOSCALING.
	AutoscalingModuleStatus *AutoscalingModuleStatus `json:"autoscalingModuleStatus,omitempty"`

	// FirewallModuleStatus: [Output Only] The status of the FirewallModule,
	// set for type FIREWALL.
	FirewallModuleStatus *FirewallModuleStatus `json:"firewallModuleStatus,omitempty"`

	// HealthCheckModuleStatus: [Output Only] The status of the
	// HealthCheckModule, set for type HEALTH_CHECK.
	HealthCheckModuleStatus *HealthCheckModuleStatus `json:"healthCheckModuleStatus,omitempty"`

	// LbModuleStatus: [Output Only] The status of the LbModule, set for
	// type LOAD_BALANCING.
	LbModuleStatus *LbModuleStatus `json:"lbModuleStatus,omitempty"`

	// NetworkModuleStatus: [Output Only] The status of the NetworkModule,
	// set for type NETWORK.
	NetworkModuleStatus *NetworkModuleStatus `json:"networkModuleStatus,omitempty"`

	// ReplicaPoolModuleStatus: [Output Only] The status of the
	// ReplicaPoolModule, set for type VM.
	ReplicaPoolModuleStatus *ReplicaPoolModuleStatus `json:"replicaPoolModuleStatus,omitempty"`

	// State: [Output Only] The current state of the module.
	State *DeployState `json:"state,omitempty"`

	// Type: [Output Only] The type of the module.
	Type string `json:"type,omitempty"`
}

type NetworkInterface struct {
	// AccessConfigs: An array of configurations for this interface. This
	// specifies how this interface is configured to interact with other
	// network services
	AccessConfigs []*AccessConfig `json:"accessConfigs,omitempty"`

	// Name: Name of the interface.
	Name string `json:"name,omitempty"`

	// Network: The name of the NetworkModule to which this interface
	// applies. If not specified, or specified as 'default', this will use
	// the 'default' network.
	Network string `json:"network,omitempty"`

	// NetworkIp: An optional IPV4 internal network address to assign to the
	// instance for this network interface.
	NetworkIp string `json:"networkIp,omitempty"`
}

type NetworkModule struct {
	// IPv4Range: Required; The range of internal addresses that are legal
	// on this network. This range is a CIDR specification, for example:
	// 192.168.0.0/16.
	IPv4Range string `json:"IPv4Range,omitempty"`

	// Description: The description of the network.
	Description string `json:"description,omitempty"`

	// GatewayIPv4: An optional address that is used for default routing to
	// other networks. This must be within the range specified by IPv4Range,
	// and is typicall the first usable address in that range. If not
	// specified, the default value is the first usable address in
	// IPv4Range.
	GatewayIPv4 string `json:"gatewayIPv4,omitempty"`
}

type NetworkModuleStatus struct {
	// NetworkUrl: [Output Only] The URL of the corresponding Network
	// resource.
	NetworkUrl string `json:"networkUrl,omitempty"`
}

type NewDisk struct {
	// Attachment: How the disk will be attached to the Replica.
	Attachment *DiskAttachment `json:"attachment,omitempty"`

	// AutoDelete: If true, then this disk will be deleted when the instance
	// is deleted.
	AutoDelete bool `json:"autoDelete,omitempty"`

	// Boot: If true, indicates that this is the root persistent disk.
	Boot bool `json:"boot,omitempty"`

	// InitializeParams: Create the new disk using these parameters. The
	// name of the disk will be <instance_name>-<five_random_charactersgt;.
	InitializeParams *NewDiskInitializeParams `json:"initializeParams,omitempty"`
}

type NewDiskInitializeParams struct {
	// DiskSizeGb: The size of the created disk in gigabytes.
	DiskSizeGb int64 `json:"diskSizeGb,omitempty,string"`

	// DiskType: Name of the disk type resource describing which disk type
	// to use to create the disk. For example 'pd-ssd' or 'pd-standard'.
	// Default is 'pd-standard'
	DiskType string `json:"diskType,omitempty"`

	// SourceImage: The fully-qualified URL of a source image to use to
	// create this disk.
	SourceImage string `json:"sourceImage,omitempty"`
}

type ParamOverride struct {
	// Path: A JSON Path expression that specifies which parameter should be
	// overridden.
	Path string `json:"path,omitempty"`

	// Value: The new value to assign to the overridden parameter.
	Value string `json:"value,omitempty"`
}

type ReplicaPoolModule struct {
	// EnvVariables: A list of environment variables.
	EnvVariables map[string]EnvVariable `json:"envVariables,omitempty"`

	// HealthChecks: The Health Checks to configure for the
	// ReplicaPoolModule
	HealthChecks []string `json:"healthChecks,omitempty"`

	// NumReplicas: Number of replicas in this module.
	NumReplicas int64 `json:"numReplicas,omitempty"`

	// ReplicaPoolParams: Information for a ReplicaPoolModule.
	ReplicaPoolParams *ReplicaPoolParams `json:"replicaPoolParams,omitempty"`

	// ResourceView: [Output Only] The name of the Resource View associated
	// with a ReplicaPoolModule. This field will be generated by the
	// service.
	ResourceView string `json:"resourceView,omitempty"`
}

type ReplicaPoolModuleStatus struct {
	// ReplicaPoolUrl: [Output Only] The URL of the associated ReplicaPool
	// resource.
	ReplicaPoolUrl string `json:"replicaPoolUrl,omitempty"`

	// ResourceViewUrl: [Output Only] The URL of the Resource Group
	// associated with this ReplicaPool.
	ResourceViewUrl string `json:"resourceViewUrl,omitempty"`
}

type ReplicaPoolParams struct {
	// V1beta1: ReplicaPoolParams specifications for use with ReplicaPools
	// v1beta1.
	V1beta1 *ReplicaPoolParamsV1Beta1 `json:"v1beta1,omitempty"`
}

type ReplicaPoolParamsV1Beta1 struct {
	// AutoRestart: Whether these replicas should be restarted if they
	// experience a failure. The default value is true.
	AutoRestart bool `json:"autoRestart,omitempty"`

	// BaseInstanceName: The base name for instances within this
	// ReplicaPool.
	BaseInstanceName string `json:"baseInstanceName,omitempty"`

	// CanIpForward: Enables IP Forwarding
	CanIpForward bool `json:"canIpForward,omitempty"`

	// Description: An optional textual description of the resource.
	Description string `json:"description,omitempty"`

	// DisksToAttach: A list of existing Persistent Disk resources to attach
	// to each replica in the pool. Each disk will be attached in read-only
	// mode to every replica.
	DisksToAttach []*ExistingDisk `json:"disksToAttach,omitempty"`

	// DisksToCreate: A list of Disk resources to create and attach to each
	// Replica in the Pool. Currently, you can only define one disk and it
	// must be a root persistent disk. Note that Replica Pool will create a
	// root persistent disk for each replica.
	DisksToCreate []*NewDisk `json:"disksToCreate,omitempty"`

	// InitAction: Name of the Action to be run during initialization of a
	// ReplicaPoolModule.
	InitAction string `json:"initAction,omitempty"`

	// MachineType: The machine type for this instance. Either a complete
	// URL, or the resource name (e.g. n1-standard-1).
	MachineType string `json:"machineType,omitempty"`

	// Metadata: The metadata key/value pairs assigned to this instance.
	Metadata *Metadata `json:"metadata,omitempty"`

	// NetworkInterfaces: A list of network interfaces for the instance.
	// Currently only one interface is supported by Google Compute Engine.
	NetworkInterfaces []*NetworkInterface `json:"networkInterfaces,omitempty"`

	OnHostMaintenance string `json:"onHostMaintenance,omitempty"`

	// ServiceAccounts: A list of Service Accounts to enable for this
	// instance.
	ServiceAccounts []*ServiceAccount `json:"serviceAccounts,omitempty"`

	// Tags: A list of tags to apply to the Google Compute Engine instance
	// to identify resources.
	Tags *Tag `json:"tags,omitempty"`

	// Zone: The zone for this ReplicaPool.
	Zone string `json:"zone,omitempty"`
}

type ServiceAccount struct {
	// Email: Service account email address.
	Email string `json:"email,omitempty"`

	// Scopes: List of OAuth2 scopes to obtain for the service account.
	Scopes []string `json:"scopes,omitempty"`
}

type Tag struct {
	// FingerPrint: The fingerprint of the tag.
	FingerPrint string `json:"fingerPrint,omitempty"`

	// Items: Items contained in this tag.
	Items []string `json:"items,omitempty"`
}

type Template struct {
	// Actions: Action definitions for use in Module intents in this
	// Template.
	Actions map[string]Action `json:"actions,omitempty"`

	// Description: A user-supplied description of this Template.
	Description string `json:"description,omitempty"`

	// Modules: A list of modules for this Template.
	Modules map[string]Module `json:"modules,omitempty"`

	// Name: Name of this Template. The name must conform to the expression:
	// [a-zA-Z0-9-_]{1,64}
	Name string `json:"name,omitempty"`
}

type TemplatesListResponse struct {
	NextPageToken string `json:"nextPageToken,omitempty"`

	Resources []*Template `json:"resources,omitempty"`
}

// method id "manager.deployments.delete":

type DeploymentsDeleteCall struct {
	s              *Service
	projectId      string
	region         string
	deploymentName string
	opt_           map[string]interface{}
}

// Delete:
func (r *DeploymentsService) Delete(projectId string, region string, deploymentName string) *DeploymentsDeleteCall {
	c := &DeploymentsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectId = projectId
	c.region = region
	c.deploymentName = deploymentName
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DeploymentsDeleteCall) Fields(s ...googleapi.Field) *DeploymentsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DeploymentsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectId}/regions/{region}/deployments/{deploymentName}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectId":      c.projectId,
		"region":         c.region,
		"deploymentName": c.deploymentName,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "",
	//   "httpMethod": "DELETE",
	//   "id": "manager.deployments.delete",
	//   "parameterOrder": [
	//     "projectId",
	//     "region",
	//     "deploymentName"
	//   ],
	//   "parameters": {
	//     "deploymentName": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "projectId": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectId}/regions/{region}/deployments/{deploymentName}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "manager.deployments.get":

type DeploymentsGetCall struct {
	s              *Service
	projectId      string
	region         string
	deploymentName string
	opt_           map[string]interface{}
}

// Get:
func (r *DeploymentsService) Get(projectId string, region string, deploymentName string) *DeploymentsGetCall {
	c := &DeploymentsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectId = projectId
	c.region = region
	c.deploymentName = deploymentName
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DeploymentsGetCall) Fields(s ...googleapi.Field) *DeploymentsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DeploymentsGetCall) Do() (*Deployment, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectId}/regions/{region}/deployments/{deploymentName}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectId":      c.projectId,
		"region":         c.region,
		"deploymentName": c.deploymentName,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Deployment
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "",
	//   "httpMethod": "GET",
	//   "id": "manager.deployments.get",
	//   "parameterOrder": [
	//     "projectId",
	//     "region",
	//     "deploymentName"
	//   ],
	//   "parameters": {
	//     "deploymentName": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "projectId": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectId}/regions/{region}/deployments/{deploymentName}",
	//   "response": {
	//     "$ref": "Deployment"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/ndev.cloudman",
	//     "https://www.googleapis.com/auth/ndev.cloudman.readonly"
	//   ]
	// }

}

// method id "manager.deployments.insert":

type DeploymentsInsertCall struct {
	s          *Service
	projectId  string
	region     string
	deployment *Deployment
	opt_       map[string]interface{}
}

// Insert:
func (r *DeploymentsService) Insert(projectId string, region string, deployment *Deployment) *DeploymentsInsertCall {
	c := &DeploymentsInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectId = projectId
	c.region = region
	c.deployment = deployment
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DeploymentsInsertCall) Fields(s ...googleapi.Field) *DeploymentsInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DeploymentsInsertCall) Do() (*Deployment, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.deployment)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectId}/regions/{region}/deployments")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectId": c.projectId,
		"region":    c.region,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Deployment
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "",
	//   "httpMethod": "POST",
	//   "id": "manager.deployments.insert",
	//   "parameterOrder": [
	//     "projectId",
	//     "region"
	//   ],
	//   "parameters": {
	//     "projectId": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectId}/regions/{region}/deployments",
	//   "request": {
	//     "$ref": "Deployment"
	//   },
	//   "response": {
	//     "$ref": "Deployment"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/appengine.admin",
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/devstorage.read_write",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "manager.deployments.list":

type DeploymentsListCall struct {
	s         *Service
	projectId string
	region    string
	opt_      map[string]interface{}
}

// List:
func (r *DeploymentsService) List(projectId string, region string) *DeploymentsListCall {
	c := &DeploymentsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectId = projectId
	c.region = region
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum count of
// results to be returned. Acceptable values are 0 to 100, inclusive.
// (Default: 50)
func (c *DeploymentsListCall) MaxResults(maxResults int64) *DeploymentsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Specifies a
// nextPageToken returned by a previous list request. This token can be
// used to request the next page of results from a previous list
// request.
func (c *DeploymentsListCall) PageToken(pageToken string) *DeploymentsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DeploymentsListCall) Fields(s ...googleapi.Field) *DeploymentsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DeploymentsListCall) Do() (*DeploymentsListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectId}/regions/{region}/deployments")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectId": c.projectId,
		"region":    c.region,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *DeploymentsListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "",
	//   "httpMethod": "GET",
	//   "id": "manager.deployments.list",
	//   "parameterOrder": [
	//     "projectId",
	//     "region"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "default": "50",
	//       "description": "Maximum count of results to be returned. Acceptable values are 0 to 100, inclusive. (Default: 50)",
	//       "format": "int32",
	//       "location": "query",
	//       "maximum": "100",
	//       "minimum": "0",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Specifies a nextPageToken returned by a previous list request. This token can be used to request the next page of results from a previous list request.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "projectId": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectId}/regions/{region}/deployments",
	//   "response": {
	//     "$ref": "DeploymentsListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/ndev.cloudman",
	//     "https://www.googleapis.com/auth/ndev.cloudman.readonly"
	//   ]
	// }

}

// method id "manager.templates.delete":

type TemplatesDeleteCall struct {
	s            *Service
	projectId    string
	templateName string
	opt_         map[string]interface{}
}

// Delete:
func (r *TemplatesService) Delete(projectId string, templateName string) *TemplatesDeleteCall {
	c := &TemplatesDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectId = projectId
	c.templateName = templateName
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TemplatesDeleteCall) Fields(s ...googleapi.Field) *TemplatesDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TemplatesDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectId}/templates/{templateName}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectId":    c.projectId,
		"templateName": c.templateName,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "",
	//   "httpMethod": "DELETE",
	//   "id": "manager.templates.delete",
	//   "parameterOrder": [
	//     "projectId",
	//     "templateName"
	//   ],
	//   "parameters": {
	//     "projectId": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "templateName": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectId}/templates/{templateName}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "manager.templates.get":

type TemplatesGetCall struct {
	s            *Service
	projectId    string
	templateName string
	opt_         map[string]interface{}
}

// Get:
func (r *TemplatesService) Get(projectId string, templateName string) *TemplatesGetCall {
	c := &TemplatesGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectId = projectId
	c.templateName = templateName
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TemplatesGetCall) Fields(s ...googleapi.Field) *TemplatesGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TemplatesGetCall) Do() (*Template, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectId}/templates/{templateName}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectId":    c.projectId,
		"templateName": c.templateName,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Template
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "",
	//   "httpMethod": "GET",
	//   "id": "manager.templates.get",
	//   "parameterOrder": [
	//     "projectId",
	//     "templateName"
	//   ],
	//   "parameters": {
	//     "projectId": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "templateName": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectId}/templates/{templateName}",
	//   "response": {
	//     "$ref": "Template"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/ndev.cloudman",
	//     "https://www.googleapis.com/auth/ndev.cloudman.readonly"
	//   ]
	// }

}

// method id "manager.templates.insert":

type TemplatesInsertCall struct {
	s         *Service
	projectId string
	template  *Template
	opt_      map[string]interface{}
}

// Insert:
func (r *TemplatesService) Insert(projectId string, template *Template) *TemplatesInsertCall {
	c := &TemplatesInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectId = projectId
	c.template = template
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TemplatesInsertCall) Fields(s ...googleapi.Field) *TemplatesInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TemplatesInsertCall) Do() (*Template, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.template)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectId}/templates")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectId": c.projectId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Template
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "",
	//   "httpMethod": "POST",
	//   "id": "manager.templates.insert",
	//   "parameterOrder": [
	//     "projectId"
	//   ],
	//   "parameters": {
	//     "projectId": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectId}/templates",
	//   "request": {
	//     "$ref": "Template"
	//   },
	//   "response": {
	//     "$ref": "Template"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "manager.templates.list":

type TemplatesListCall struct {
	s         *Service
	projectId string
	opt_      map[string]interface{}
}

// List:
func (r *TemplatesService) List(projectId string) *TemplatesListCall {
	c := &TemplatesListCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectId = projectId
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum count of
// results to be returned. Acceptable values are 0 to 100, inclusive.
// (Default: 50)
func (c *TemplatesListCall) MaxResults(maxResults int64) *TemplatesListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Specifies a
// nextPageToken returned by a previous list request. This token can be
// used to request the next page of results from a previous list
// request.
func (c *TemplatesListCall) PageToken(pageToken string) *TemplatesListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TemplatesListCall) Fields(s ...googleapi.Field) *TemplatesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TemplatesListCall) Do() (*TemplatesListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectId}/templates")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectId": c.projectId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *TemplatesListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "",
	//   "httpMethod": "GET",
	//   "id": "manager.templates.list",
	//   "parameterOrder": [
	//     "projectId"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "default": "50",
	//       "description": "Maximum count of results to be returned. Acceptable values are 0 to 100, inclusive. (Default: 50)",
	//       "format": "int32",
	//       "location": "query",
	//       "maximum": "100",
	//       "minimum": "0",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Specifies a nextPageToken returned by a previous list request. This token can be used to request the next page of results from a previous list request.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "projectId": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectId}/templates",
	//   "response": {
	//     "$ref": "TemplatesListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/ndev.cloudman",
	//     "https://www.googleapis.com/auth/ndev.cloudman.readonly"
	//   ]
	// }

}
