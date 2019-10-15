package couchbase

import (
	"fmt"
	"regexp"
	"time"

	"gopkg.in/couchbase/gocb.v1"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
)

const (
	quotaBucket          = 600
	dexType              = "dex_type"
	dexSecondaryIndexPre = "idx_dex"
	dexIndexTypeVersion  = "v01"
	primaryIndex         = "#primary"
)

var BucketName string = "dex"

// NetworkDB contains options to couchbase database accessed over network.
type NetworkDB struct {
	Bucket            string
	User              string
	Password          string
	Host              string
	Port              uint16
	QuotaRam          int  // default:0
	CreateIfNotExists bool // default: false
}

//not fully tested
type SSL struct {
	CertFile string
}

// Couchbase options for creating a couchbase bucket
type Couchbase struct {
	NetworkDB
	SSL SSL `json:"ssl" yaml:"ssl"`
}

var strEsc = regexp.MustCompile(`([\\'])`)

func dataSourceStr(str string) string {
	return strEsc.ReplaceAllString(str, `\$1`)
}

// Open creates a new storage implementation backed by Couchbase.
func (cb *Couchbase) Open(logger log.Logger) (storage.Storage, error) {
	conn, err := cb.open(logger)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (cb *Couchbase) create_bucket(logger log.Logger, bucket_name string, cluster_manager *gocb.ClusterManager) error {
	quotaRam := cb.QuotaRam
	if quotaRam == 0 {
		quotaRam = quotaBucket
	}
	bucketSettings := gocb.BucketSettings{
		FlushEnabled:  false,
		IndexReplicas: false,
		Name:          bucket_name,
		Password:      "",
		Quota:         quotaRam,
		Replicas:      1,
	}
	err := cluster_manager.InsertBucket(&bucketSettings)
	if err != nil {
		logger.Errorf(err.Error())
		return err
	}
	return nil
}

func (cb *Couchbase) get_connection_string() string {
	connection_string := fmt.Sprintf("couchbase://%s", dataSourceStr(cb.Host))
	if cb.SSL.CertFile != "" {
		connection_string = fmt.Sprintf("couchbases://%s?certpath=%s", dataSourceStr(cb.Host), cb.SSL.CertFile)
	}
	return connection_string
}

func (cb *Couchbase) open(logger log.Logger) (*conn, error) {
	connection_string := cb.get_connection_string()
	cb_cluster, err := gocb.Connect(connection_string)
	if err != nil {
		return nil, err
	}
	cb_cluster.Authenticate(gocb.PasswordAuthenticator{
		Username: cb.User,
		Password: cb.Password,
	})
	cluster_manager := cb_cluster.Manager(cb.User, cb.Password)

	// Verify if a bucket with that name already exists
	list_buckets, err_list_bucket := cluster_manager.GetBuckets()
	BucketName = cb.Bucket
	already_exists := false
	if err_list_bucket == nil {
		for _, bucket_var := range list_buckets {
			if bucket_var.Name != "" && bucket_var.Name == BucketName {
				already_exists = true
				break
			}
		}

	}
	if !already_exists {
		if cb.CreateIfNotExists {
			err := cb.create_bucket(logger, BucketName, cluster_manager)
			if err != nil {
				return nil, err
			}
			// wait because indexex have to be created
			time.Sleep(3 * time.Second)
		} else {
			return nil, fmt.Errorf("The bucket %s does not exist", BucketName)
		}
	}
	bucket, err := cb_cluster.OpenBucket(BucketName, "")

	// create indexes
	bucket_manager := bucket.Manager(cb.User, cb.Password)
	bucket_manager.CreatePrimaryIndex(primaryIndex, true, false)
	bucket_manager.CreateIndex(dexSecondaryIndexPre+"_"+dexType+"_"+dexIndexTypeVersion, []string{dexType}, true, false)

	c := &conn{
		db:     bucket,
		logger: logger,
	}
	return c, nil
}
