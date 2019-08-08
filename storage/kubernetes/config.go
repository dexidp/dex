package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

const (
	envK8sServiceHost  = "KUBERNETES_SERVICE_HOST"
	envK8sServicePort  = "KUBERNETES_SERVICE_PORT"
	envK8sPodNamespace = "KUBERNETES_POD_NAMESPACE"

	k8sCACertificatePath       = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	k8sServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	k8sDefaultNamespace        = "default"
)

// Config values for the Kubernetes storage type.
type Config struct {
	InCluster      bool   `json:"inCluster"`
	KubeConfigFile string `json:"kubeConfigFile"`
}

// Open returns a storage using Kubernetes third party resource.
func (c *Config) Open(logger log.Logger) (storage.Storage, error) {
	cli, err := c.open(logger, false)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// open returns a kubernetes client, initializing the third party resources used
// by dex.
//
// waitForResources controls if errors creating the resources cause this method to return
// immediately (used during testing), or if the client will asynchronously retry.
func (c *Config) open(logger log.Logger, waitForResources bool) (*client, error) {
	if c.InCluster && (c.KubeConfigFile != "") {
		logger.Error(errSpecifiedBothKubeConfigAndInCluster)
		return nil, storage.Error{Code: storage.ErrStorageMisconfigured, Details: errSpecifiedBothKubeConfigAndInCluster}
	}
	if !c.InCluster && (c.KubeConfigFile == "") {
		logger.Error(errSpecifiedNeitherKubeConfigAndInCluster)
		return nil, storage.Error{Code: storage.ErrStorageMisconfigured, Details: errSpecifiedNeitherKubeConfigAndInCluster}
	}

	var (
		cluster   k8sapi.Cluster
		user      k8sapi.AuthInfo
		namespace string
		err       error
	)
	if c.InCluster {
		cluster, user, namespace, err = inClusterConfig()
	} else {
		cluster, user, namespace, err = loadKubeConfig(c.KubeConfigFile)
	}
	if err != nil {
		logger.Errorf("could not create a kubernetes client, error was: %v", err)
		return nil, storage.Error{Code: storage.ErrStorageMisconfigured, Details: errCouldNotCreateK8sClient}
	}

	cli, err := newClient(cluster, user, namespace, logger)
	if err != nil {
		logger.Errorf("create client: %v", err)
		return nil, storage.Error{Code: storage.ErrStorageMisconfigured, Details: errCouldNotCreateK8sClient}
	}

	ctx, cancel := context.WithCancel(context.Background())

	//TODO This might not be best form - dex should not require the ability to create/update CRDs on a cluster, as this
	// is largely a cluster administrator privilege and we should not need to be a cluster administrator to run dex.
	// Consider making this being run as an option `createCustomResources = true` in the configuration? - @venezia
	logger.Info("creating custom Kubernetes resources")
	if !cli.registerCustomResources() {
		if waitForResources {
			cancel()
			logger.Error(errCouldNotCreateCRD)
			return nil, storage.Error{Code: storage.ErrStorageMisconfigured, Details: errCouldNotCreateCRD}
		}

		// Try to synchronously create the custom resources once. This doesn't mean
		// they'll immediately be available, but ensures that the client will actually try
		// once.
		logger.Errorf("%s: %v", errCouldNotCreateCRD, err)
		go func() {
			for {
				if cli.registerCustomResources() {
					return
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
				}
			}
		}()
	}

	if waitForResources {
		if err := cli.waitForCRDs(ctx); err != nil {
			cancel()
			return nil, storage.Error{Code: storage.ErrStorageMisconfigured, Details: errCRDsNotReady}
		}
	}

	// If the client is closed, stop trying to create resources.
	cli.cancel = cancel
	return cli, nil
}

func loadKubeConfig(kubeConfigPath string) (cluster k8sapi.Cluster, user k8sapi.AuthInfo, namespace string, err error) {
	data, err := ioutil.ReadFile(kubeConfigPath)
	if err != nil {
		err = fmt.Errorf("read %s: %v", kubeConfigPath, err)
		return
	}

	var c k8sapi.Config
	if err = yaml.Unmarshal(data, &c); err != nil {
		err = fmt.Errorf("unmarshal %s: %v", kubeConfigPath, err)
		return
	}

	cluster, user, namespace, err = currentContext(&c)
	if namespace == "" {
		namespace = k8sDefaultNamespace
	}
	return
}

func currentContext(config *k8sapi.Config) (cluster k8sapi.Cluster, user k8sapi.AuthInfo, ns string, err error) {
	if config.CurrentContext == "" {
		if len(config.Contexts) == 1 {
			config.CurrentContext = config.Contexts[0].Name
		} else {
			return cluster, user, "", errors.New("kubeconfig has no current context")
		}
	}
	context, ok := func() (k8sapi.Context, bool) {
		for _, namedContext := range config.Contexts {
			if namedContext.Name == config.CurrentContext {
				return namedContext.Context, true
			}
		}
		return k8sapi.Context{}, false
	}()
	if !ok {
		return cluster, user, "", fmt.Errorf("no context named %q found", config.CurrentContext)
	}

	cluster, ok = func() (k8sapi.Cluster, bool) {
		for _, namedCluster := range config.Clusters {
			if namedCluster.Name == context.Cluster {
				return namedCluster.Cluster, true
			}
		}
		return k8sapi.Cluster{}, false
	}()
	if !ok {
		return cluster, user, "", fmt.Errorf("no cluster named %q found", context.Cluster)
	}

	user, ok = func() (k8sapi.AuthInfo, bool) {
		for _, namedAuthInfo := range config.AuthInfos {
			if namedAuthInfo.Name == context.AuthInfo {
				return namedAuthInfo.AuthInfo, true
			}
		}
		return k8sapi.AuthInfo{}, false
	}()
	if !ok {
		return cluster, user, "", fmt.Errorf("no user named %q found", context.AuthInfo)
	}
	return cluster, user, context.Namespace, nil
}

func inClusterConfig() (cluster k8sapi.Cluster, user k8sapi.AuthInfo, namespace string, err error) {
	host, port := os.Getenv(envK8sServiceHost), os.Getenv(envK8sServicePort)
	if len(host) == 0 || len(port) == 0 {
		err = fmt.Errorf("unable to load in-cluster configuration, %s and %s must be defined", envK8sServiceHost, envK8sServicePort)
		return
	}
	cluster = k8sapi.Cluster{
		Server:               "https://" + host + ":" + port,
		CertificateAuthority: k8sCACertificatePath,
	}
	token, err := ioutil.ReadFile(k8sServiceAccountTokenPath)
	if err != nil {
		return
	}
	user = k8sapi.AuthInfo{Token: string(token)}

	if namespace = os.Getenv(envK8sPodNamespace); namespace == "" {
		namespace, err = namespaceFromServiceAccountJWT(user.Token)
		if err != nil {
			err = fmt.Errorf("failed to inspect service account token: %v", err)
			return
		}
	}

	return
}

func namespaceFromServiceAccountJWT(s string) (string, error) {
	// The service account token is just a JWT. Parse it as such.
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		// It's extremely important we don't log the actual service account token.
		return "", fmt.Errorf("malformed service account token: expected 3 parts got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("malformed service account token: %v", err)
	}
	var data struct {
		// The claim Kubernetes uses to identify which namespace a service account belongs to.
		//
		// See: https://github.com/kubernetes/kubernetes/blob/v1.4.3/pkg/serviceaccount/jwt.go#L42
		Namespace string `json:"kubernetes.io/serviceaccount/namespace"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", fmt.Errorf("malformed service account token: %v", err)
	}
	if data.Namespace == "" {
		return "", errors.New(`jwt claim "kubernetes.io/serviceaccount/namespace" not found`)
	}
	return data.Namespace, nil
}

// registerCustomResources attempts to create the custom resources dex
// requires or identifies that they're already enabled. This function creates
// custom resource definitions(CRDs)
// It logs all errors, returning true if the resources were created successfully.
//
// Creating a custom resource does not mean that they'll be immediately available.
func (cli *client) registerCustomResources() (ok bool) {
	ok = true
	length := len(customResourceDefinitions)
	for i := 0; i < length; i++ {
		var err error
		var resourceName string

		r := customResourceDefinitions[i]
		var i interface{}
		cli.logger.Infof("checking if custom resource %s has been created already...", r.ObjectMeta.Name)
		if err := cli.list(r.Spec.Names.Plural, &i); err == nil {
			cli.logger.Infof("The custom resource %s already available, skipping create", r.ObjectMeta.Name)
			continue
		} else {
			cli.logger.Infof("failed to list custom resource %s, attempting to create: %v", r.ObjectMeta.Name, err)
		}
		err = cli.postResource(customResourceDefinitionAPI, customResourceDefinitionNamespace, customResourceDefinitionResource, r)
		resourceName = r.ObjectMeta.Name

		if err != nil {
			switch t := err.(type) {
			case storage.Error:
				switch t.Code {
				case storage.ErrAlreadyExists:
					cli.logger.Infof("custom resource definition already created %s", resourceName)
				case storage.ErrNotFound:
					cli.logger.Errorf("custom resource definition not found, please enable the respective API group")
					ok = false
				default:
					cli.logger.Errorf("creating custom resource %s: %v", resourceName, err)
					ok = false
				}
			default:
				cli.logger.Errorf("creating custom resource %s: %v", resourceName, err)
				ok = false
			}
			continue
		}
		cli.logger.Errorf("create custom resource %s", resourceName)
	}
	return ok
}

// waitForCRDs waits for all CRDs to be in a ready state, and is used
// by the tests to synchronize before running conformance.
func (cli *client) waitForCRDs(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	for _, crd := range customResourceDefinitions {
		for {
			err := cli.isCRDReady(crd.Name)
			if err == nil {
				break
			}

			cli.logger.Errorf("checking CRD: %v", err)

			select {
			case <-ctx.Done():
				return errors.New("timed out waiting for CRDs to be available")
			case <-time.After(time.Millisecond * 100):
			}
		}
	}
	return nil
}

// isCRDReady determines if a CRD is ready by inspecting its conditions.
func (cli *client) isCRDReady(name string) error {
	var r k8sapi.CustomResourceDefinition
	err := cli.getResource(customResourceDefinitionAPI, customResourceDefinitionNamespace, customResourceDefinitionResource, name, &r)
	if err != nil {
		return fmt.Errorf("get crd %s: %v", name, err)
	}

	conds := make(map[string]string) // For debugging, keep the conditions around.
	for _, c := range r.Status.Conditions {
		if c.Type == k8sapi.Established && c.Status == k8sapi.ConditionTrue {
			return nil
		}
		conds[string(c.Type)] = string(c.Status)
	}
	return fmt.Errorf("crd %s not ready %#v", name, conds)
}
