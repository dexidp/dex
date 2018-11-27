package keystone

import (
	"testing"
	"github.com/dexidp/dex/connector"

	"fmt"
	"io"
	"os"
  	"time"
  	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
   	networktypes "github.com/docker/docker/api/types/network"
  	"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"
	"bytes"
	"encoding/json"
	"io/ioutil"
)

const dockerCliVersion = "1.37"

const exposedKeystonePort = "5000"
const exposedKeystonePortAdmin = "35357"

const keystoneHost = "http://localhost"
const keystoneURL = keystoneHost + ":" + exposedKeystonePort
const keystoneAdminURL = keystoneHost + ":" + exposedKeystonePortAdmin
const authTokenURL = keystoneURL + "/v3/auth/tokens/"
const userURL = keystoneAdminURL + "/v3/users/"
const groupURL = keystoneAdminURL + "/v3/groups/"

func startKeystoneContainer() string {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion(dockerCliVersion))

	if err != nil {
    	fmt.Printf("Error %v", err)
		return ""
	}

	imageName := "openio/openstack-keystone"
	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
    	fmt.Printf("Error %v", err)
		return ""
	}
	io.Copy(os.Stdout, out)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
    }, &container.HostConfig{
    		PortBindings: nat.PortMap{
        		"5000/tcp": []nat.PortBinding{
            		{
                		HostIP:   "0.0.0.0",
                		HostPort: exposedKeystonePort,
            		},
        		},
				"35357/tcp": []nat.PortBinding{
					{
					HostIP:   "0.0.0.0",
					HostPort: exposedKeystonePortAdmin,
					},
				},
    		},
		}, &networktypes.NetworkingConfig{}, "dex_keystone_test")

	if err != nil {
    	fmt.Printf("Error %v", err)
		return ""
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	fmt.Println(resp.ID)
  	return resp.ID
}

func cleanKeystoneContainer(ID string) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion(dockerCliVersion))
	if err != nil {
		fmt.Printf("Error %v", err)
		return
	}
	duration := time.Duration(1)
	if err:= cli.ContainerStop(ctx, ID, &duration); err != nil {
		fmt.Printf("Error %v", err)
		return
	}
	if err:= cli.ContainerRemove(ctx, ID, types.ContainerRemoveOptions{}); err != nil {
		fmt.Printf("Error %v", err)
	}
}

func getAdminToken(admin_name, admin_pass string) (token string) {
	client := &http.Client{}

	jsonData := LoginRequestData{
		Auth: Auth{
			Identity: Identity{
				Methods:[]string{"password"},
				Password: Password{
					User: User{
						Name: admin_name,
						Domain: Domain{ID: "default"},
						Password: admin_pass,
					},
				},
			},
		},
	}

	body, _ := json.Marshal(jsonData)

	req, _ := http.NewRequest("POST", authTokenURL, bytes.NewBuffer(body))

	req.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(req)

	token = resp.Header["X-Subject-Token"][0]
	return token
}

func createUser(token, user_name, user_email, user_pass string) (string){
	client := &http.Client{}

	createUserData := CreateUserRequest{
		CreateUser: CreateUserForm{
			Name: user_name,
			Email: user_email,
			Enabled: true,
			Password: user_pass,
			Roles: []string{"admin"},
		},
	}

	body, _ := json.Marshal(createUserData)

	req, _ := http.NewRequest("POST", userURL, bytes.NewBuffer(body))
	req.Header.Set("X-Auth-Token", token)
	req.Header.Add("Content-Type", "application/json")
	resp, _ := client.Do(req)

	data, _ := ioutil.ReadAll(resp.Body)
	var userResponse = new(UserResponse)
	err := json.Unmarshal(data, &userResponse)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(userResponse.User.ID)
	return userResponse.User.ID

}

func deleteUser(token, id string) {
	client := &http.Client{}

	deleteUserURI := userURL + id
	fmt.Println(deleteUserURI)
	req, _ := http.NewRequest("DELETE", deleteUserURI, nil)
	req.Header.Set("X-Auth-Token", token)
	resp, _ := client.Do(req)
	fmt.Println(resp)
}

func createGroup(token, description, name string) string{
	client := &http.Client{}

	createGroupData := CreateGroup{
		CreateGroupForm{
			Description: description,
			Name: name,
		},
	}

	body, _ := json.Marshal(createGroupData)

	req, _ := http.NewRequest("POST", groupURL, bytes.NewBuffer(body))
	req.Header.Set("X-Auth-Token", token)
	req.Header.Add("Content-Type", "application/json")
	resp, _ := client.Do(req)
	data, _ := ioutil.ReadAll(resp.Body)

	var groupResponse = new(GroupID)
	err := json.Unmarshal(data, &groupResponse)
	if err != nil {
		fmt.Println(err)
	}

	return groupResponse.Group.ID
}

func addUserToGroup(token, groupId, userId string) {
	uri := groupURL + groupId + "/users/" + userId
	client := &http.Client{}
	req, _ := http.NewRequest("PUT", uri, nil)
	req.Header.Set("X-Auth-Token", token)
	resp, _ := client.Do(req)
	fmt.Println(resp)
}

const adminUser = "demo"
const adminPass = "DEMO_PASS"
const invalidPass = "WRONG_PASS"

const testUser = "test_user"
const testPass = "test_pass"
const testEmail = "test@example.com"

const domain = "default"

func TestIncorrectCredentialsLogin(t *testing.T) {
  	c := Connector{KeystoneHost: keystoneURL, Domain: domain,
  				   KeystoneUsername: adminUser, KeystonePassword: adminPass}
  	s := connector.Scopes{OfflineAccess: true, Groups: true}
  	_, validPW, _ := c.Login(context.Background(), s, adminUser, invalidPass)

  	if validPW {
  		t.Fail()
  	}
}

func TestValidUserLogin(t *testing.T) {
	token := getAdminToken(adminUser, adminPass)
	userID := createUser(token, testUser, testEmail, testPass)
  	c := Connector{KeystoneHost: keystoneURL, Domain: domain,
  				  KeystoneUsername: adminUser, KeystonePassword: adminPass}
  	s := connector.Scopes{OfflineAccess: true, Groups: true}
  	_, validPW, _ := c.Login(context.Background(), s, testUser, testPass)
  	if !validPW {
     	t.Fail()
  	}
  	deleteUser(token, userID)
}

func TestUseRefreshToken(t *testing.T) {
  t.Fatal("Not implemented")
}

func TestUseRefreshTokenUserDeleted(t *testing.T){
  t.Fatal("Not implemented")
}

func TestUseRefreshTokenGroupsChanged(t *testing.T){
	t.Fatal("Not implemented")
}

func TestMain(m *testing.M) {
	dockerID := startKeystoneContainer()
  	repeats := 10
  	running := false
  	for i := 0; i < repeats; i++ {
   		_, err := http.Get(keystoneURL)
   		if err == nil {
     		running = true
     		break
   		}
   		time.Sleep(10 * time.Second)
  	}
  	if !running {
    	fmt.Printf("Failed to start keystone container")
    	os.Exit(1)
  	}
  	defer cleanKeystoneContainer(dockerID)
  	// run all tests
	m.Run()
}
