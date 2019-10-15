package gocb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
)

// ClusterManager provides methods for performing cluster management operations.
type ClusterManager struct {
	hosts    []string
	username string
	password string
	httpCli  *http.Client
	cluster  *Cluster
}

// BucketType specifies the kind of bucket
type BucketType int

const (
	// Couchbase indicates a Couchbase bucket type.
	Couchbase = BucketType(0)

	// Memcached indicates a Memcached bucket type.
	Memcached = BucketType(1)

	// Ephemeral indicates an Ephemeral bucket type.
	Ephemeral = BucketType(2)
)

type bucketDataIn struct {
	Name         string `json:"name"`
	BucketType   string `json:"bucketType"`
	AuthType     string `json:"authType"`
	SaslPassword string `json:"saslPassword"`
	Quota        struct {
		Ram    int `json:"ram"`
		RawRam int `json:"rawRAM"`
	} `json:"quota"`
	ReplicaNumber int  `json:"replicaNumber"`
	ReplicaIndex  bool `json:"replicaIndex"`
	Controllers   struct {
		Flush string `json:"flush"`
	} `json:"controllers"`
}

// NodeMetadata contains information about a node in the cluster.
type NodeMetadata struct {
	ClusterCompatibility int                `json:"clusterCompatibility"`
	ClusterMembership    string             `json:"clusterMembership"`
	CouchAPIBase         string             `json:"couchApiBase"`
	Hostname             string             `json:"hostname"`
	InterestingStats     map[string]float64 `json:"interestingStats,omitempty"`
	MCDMemoryAllocated   float64            `json:"mcdMemoryAllocated"`
	MCDMemoryReserved    float64            `json:"mcdMemoryReserved"`
	MemoryFree           float64            `json:"memoryFree"`
	MemoryTotal          float64            `json:"memoryTotal"`
	OS                   string             `json:"os"`
	Ports                map[string]int     `json:"ports"`
	Status               string             `json:"status"`
	Uptime               int                `json:"uptime,string"`
	Version              string             `json:"version"`
	ThisNode             bool               `json:"thisNode,omitempty"`
}

// clusterCfg contains information about the cluster setup, we only need a subset of that information.
type clusterCfg struct {
	Nodes []NodeMetadata `json:"nodes"`
}

// BucketSettings holds information about the settings for a bucket.
type BucketSettings struct {
	FlushEnabled  bool
	IndexReplicas bool
	Name          string
	Password      string
	Quota         int
	Replicas      int
	Type          BucketType
}

func (cm *ClusterManager) getMgmtEp() string {
	return cm.hosts[rand.Intn(len(cm.hosts))]
}

func (cm *ClusterManager) mgmtRequest(method, uri string, contentType string, body io.Reader) (*http.Response, error) {
	if contentType == "" && body != nil {
		panic("Content-type must be specified for non-null body.")
	}

	reqUri := cm.getMgmtEp() + uri
	req, err := http.NewRequest(method, reqUri, body)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}
	if cm.username != "" || cm.password != "" {
		req.SetBasicAuth(cm.username, cm.password)
	}

	return cm.httpCli.Do(req)
}

func bucketDataInToSettings(bucketData *bucketDataIn) *BucketSettings {
	settings := &BucketSettings{
		FlushEnabled:  bucketData.Controllers.Flush != "",
		IndexReplicas: bucketData.ReplicaIndex,
		Name:          bucketData.Name,
		Password:      bucketData.SaslPassword,
		Quota:         bucketData.Quota.Ram,
		Replicas:      bucketData.ReplicaNumber,
	}
	if bucketData.BucketType == "membase" {
		settings.Type = Couchbase
	} else if bucketData.BucketType == "memcached" {
		settings.Type = Memcached
	} else if bucketData.BucketType == "ephemeral" {
		settings.Type = Ephemeral
	} else {
		panic("Unrecognized bucket type string.")
	}
	if bucketData.AuthType != "sasl" {
		settings.Password = ""
	}
	return settings
}

// GetBuckets returns a list of all active buckets on the cluster.
func (cm *ClusterManager) GetBuckets() ([]*BucketSettings, error) {
	resp, err := cm.mgmtRequest("GET", "/pools/default/buckets", "", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		err = resp.Body.Close()
		if err != nil {
			logDebugf("Failed to close socket (%s)", err)
		}
		return nil, clientError{string(data)}
	}

	var bucketsData []*bucketDataIn
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&bucketsData)
	if err != nil {
		return nil, err
	}

	var buckets []*BucketSettings
	for _, bucketData := range bucketsData {
		buckets = append(buckets, bucketDataInToSettings(bucketData))
	}

	return buckets, nil
}

// InsertBucket creates a new bucket on the cluster.
func (cm *ClusterManager) InsertBucket(settings *BucketSettings) error {
	posts := url.Values{}
	posts.Add("name", settings.Name)
	if settings.Type == Couchbase {
		posts.Add("bucketType", "couchbase")
	} else if settings.Type == Memcached {
		posts.Add("bucketType", "memcached")
	} else if settings.Type == Ephemeral {
		posts.Add("bucketType", "ephemeral")
	} else {
		panic("Unrecognized bucket type.")
	}
	if settings.FlushEnabled {
		posts.Add("flushEnabled", "1")
	} else {
		posts.Add("flushEnabled", "0")
	}
	posts.Add("replicaNumber", fmt.Sprintf("%d", settings.Replicas))
	posts.Add("authType", "sasl")
	posts.Add("saslPassword", settings.Password)
	posts.Add("ramQuotaMB", fmt.Sprintf("%d", settings.Quota))

	data := []byte(posts.Encode())
	resp, err := cm.mgmtRequest("POST", "/pools/default/buckets", "application/x-www-form-urlencoded", bytes.NewReader(data))
	if err != nil {
		return err
	}

	if resp.StatusCode != 202 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = resp.Body.Close()
		if err != nil {
			logDebugf("Failed to close socket (%s)", err)
		}
		return clientError{string(data)}
	}

	return nil
}

// UpdateBucket will update the settings for a specific bucket on the cluster.
func (cm *ClusterManager) UpdateBucket(settings *BucketSettings) error {
	// Cluster-side, updates are the same as creates.
	return cm.InsertBucket(settings)
}

// RemoveBucket will delete a bucket from the cluster by name.
func (cm *ClusterManager) RemoveBucket(name string) error {
	reqUri := fmt.Sprintf("/pools/default/buckets/%s", name)

	resp, err := cm.mgmtRequest("DELETE", reqUri, "", nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = resp.Body.Close()
		if err != nil {
			logDebugf("Failed to close socket (%s)", err)
		}
		return clientError{string(data)}
	}

	return nil
}

// UserRole represents a role for a particular user on the server.
type UserRole struct {
	Role       string
	BucketName string
}

// User represents a user which was retrieved from the server.
type User struct {
	Id    string
	Name  string
	Type  string
	Roles []UserRole
}

// AuthDomain specifies the user domain of a specific user
type AuthDomain string

const (
	// LocalDomain specifies users that are locally stored in Couchbase.
	LocalDomain AuthDomain = "local"

	// ExternalDomain specifies users that are externally stored
	// (in LDAP for instance).
	ExternalDomain = "external"
)

// UserSettings represents a user during user creation.
type UserSettings struct {
	Name     string
	Password string
	Roles    []UserRole
}

type userRoleJson struct {
	Role       string `json:"role"`
	BucketName string `json:"bucket_name"`
}

type userJson struct {
	Id    string         `json:"id"`
	Name  string         `json:"name"`
	Type  string         `json:"type"`
	Roles []userRoleJson `json:"roles"`
}

type userSettingsJson struct {
	Name     string         `json:"name"`
	Password string         `json:"password"`
	Roles    []userRoleJson `json:"roles"`
}

// ClusterManagerInternal holds internally used cluster manager extension methods.
//
// Internal: This should never be used and is not supported.
type ClusterManagerInternal struct {
	manager *ClusterManager
}

func transformUserJson(userData *userJson) User {
	var user User
	user.Id = userData.Id
	user.Name = userData.Name
	user.Type = userData.Type
	for _, roleData := range userData.Roles {
		user.Roles = append(user.Roles, UserRole{
			Role:       roleData.Role,
			BucketName: roleData.BucketName,
		})
	}
	return user
}

// GetUsers returns a list of all users on the cluster.
func (cm *ClusterManager) GetUsers(domain AuthDomain) ([]*User, error) {
	uri := fmt.Sprintf("/settings/rbac/users/%s", domain)
	resp, err := cm.mgmtRequest("GET", uri, "", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		err = resp.Body.Close()
		if err != nil {
			logDebugf("Failed to close socket (%s)", err)
		}
		return nil, clientError{string(data)}
	}

	var usersData []*userJson
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&usersData)
	if err != nil {
		return nil, err
	}

	var users []*User
	for _, userData := range usersData {
		user := transformUserJson(userData)
		users = append(users, &user)
	}

	return users, nil
}

// GetUser returns the data for a particular user
func (cm *ClusterManager) GetUser(domain AuthDomain, name string) (*User, error) {
	uri := fmt.Sprintf("/settings/rbac/users/%s/%s", domain, name)
	resp, err := cm.mgmtRequest("GET", uri, "", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		err = resp.Body.Close()
		if err != nil {
			logDebugf("Failed to close socket (%s)", err)
		}
		return nil, clientError{string(data)}
	}

	var userData userJson
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&userData)
	if err != nil {
		return nil, err
	}

	user := transformUserJson(&userData)
	return &user, nil
}

// UpsertUser updates a built-in RBAC user on the cluster.
func (cm *ClusterManager) UpsertUser(domain AuthDomain, name string, settings *UserSettings) error {
	var reqRoleStrs []string
	for _, roleData := range settings.Roles {
		reqRoleStrs = append(reqRoleStrs, fmt.Sprintf("%s[%s]", roleData.Role, roleData.BucketName))
	}

	reqForm := make(url.Values)
	reqForm.Add("name", settings.Name)
	reqForm.Add("password", settings.Password)
	reqForm.Add("roles", strings.Join(reqRoleStrs, ","))

	uri := fmt.Sprintf("/settings/rbac/users/%s/%s", domain, name)
	reqBody := bytes.NewReader([]byte(reqForm.Encode()))
	resp, err := cm.mgmtRequest("PUT", uri, "application/x-www-form-urlencoded", reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = resp.Body.Close()
		if err != nil {
			logDebugf("Failed to close socket (%s)", err)
		}
		return clientError{string(data)}
	}

	return nil
}

// RemoveUser removes a built-in RBAC user on the cluster.
func (cm *ClusterManager) RemoveUser(domain AuthDomain, name string) error {
	uri := fmt.Sprintf("/settings/rbac/users/%s/%s", domain, name)
	resp, err := cm.mgmtRequest("DELETE", uri, "", nil)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = resp.Body.Close()
		if err != nil {
			logDebugf("Failed to close socket (%s)", err)
		}
		return clientError{string(data)}
	}

	return nil
}

// SearchIndexManager returns a SearchIndexManager for performing FTS index management on this cluster
// Experimental: This API is subject to change at any time.
func (cm *ClusterManager) SearchIndexManager() *SearchIndexManager {
	return &SearchIndexManager{
		authenticator: cm.cluster.auth,
		httpCli:       cm.httpCli,
		cluster:       cm.cluster,
	}
}

// Internal returns a ClusterManagerInternal internally used cluster manager extension methods.
//
// Internal: This should never be used and is not supported.
func (cm *ClusterManager) Internal() *ClusterManagerInternal {
	return &ClusterManagerInternal{
		manager: cm,
	}
}

// GetNodesMetadata returns a list of information about nodes in the cluster.
func (cmi *ClusterManagerInternal) GetNodesMetadata() ([]NodeMetadata, error) {
	resp, err := cmi.manager.mgmtRequest("GET", "/pools/default", "", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		err = resp.Body.Close()
		if err != nil {
			logDebugf("Failed to close socket (%s)", err)
		}
		return nil, clientError{string(data)}
	}

	var clusterData clusterCfg
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&clusterData)
	if err != nil {
		return nil, err
	}

	return clusterData.Nodes, nil
}
