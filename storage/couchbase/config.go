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
	quotaBucket           = 600
	dexType               = "dex_type"
	dexSecondaryIndexName = "idx_dex_dex_type_v01"
)

var BucketName string
var createBucketIfNotExists bool = false
var CreateSecondaryIndex bool = false
var CreatePrimaryIndex bool = false

// NetworkDB contains options to couchbase database accessed over network.
type NetworkDB struct {
	Bucket   string
	User     string
	Password string
	Host     string
	Port     uint16
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

func (cb *Couchbase) create_bucket(logger log.Logger, bucket_name string, cb_cluster *gocb.Cluster) error {
	cluster_manager := cb_cluster.Manager(cb.User, cb.Password)
	bucketSettings := gocb.BucketSettings{
		FlushEnabled:  false,
		IndexReplicas: false,
		Name:          bucket_name,
		Password:      "",
		Quota:         quotaBucket,
		Replicas:      1,
	}
	err := cluster_manager.InsertBucket(&bucketSettings)
	if err != nil {
		logger.Errorf(err.Error())
		return err
	}
	time.Sleep(3 * time.Second)
	return nil
}

func (cb *Couchbase) bucket_exists(cb_cluster *gocb.Cluster) bool {
	cluster_manager := cb_cluster.Manager(cb.User, cb.Password)
	list_buckets, err_list_bucket := cluster_manager.GetBuckets()
	exists := false
	if err_list_bucket == nil {
		for _, bucket_var := range list_buckets {
			if bucket_var.Name != "" && bucket_var.Name == BucketName {
				exists = true
				break
			}
		}

	}
	return exists
}

func (cb *Couchbase) create_indexes(bucket *gocb.Bucket) {
	// create indexes
	if CreatePrimaryIndex || CreateSecondaryIndex {
		bucket_manager := bucket.Manager(cb.User, cb.Password)
		if CreatePrimaryIndex {
			bucket_manager.CreatePrimaryIndex("#primary", true, false)
		}
		if CreateSecondaryIndex {
			bucket_manager.CreateIndex(dexSecondaryIndexName, []string{dexType}, true, false)
		}
	}
}

func (cb *Couchbase) get_connection_string() string {
	connection_string := fmt.Sprintf("couchbase://%s", dataSourceStr(cb.Host))
	if cb.SSL.CertFile != "" {
		connection_string = fmt.Sprintf("couchbases://%s?certpath=%s", dataSourceStr(cb.Host), cb.SSL.CertFile)
	}
	return connection_string
}

func (cb *Couchbase) open(logger log.Logger) (*conn, error) {
	BucketName = cb.Bucket
	connection_string := cb.get_connection_string()
	cb_cluster, err := gocb.Connect(connection_string)
	if err != nil {
		return nil, err
	}
	cb_cluster.Authenticate(gocb.PasswordAuthenticator{
		Username: cb.User,
		Password: cb.Password,
	})
	if createBucketIfNotExists && !cb.bucket_exists(cb_cluster) {
		cb.create_bucket(logger, BucketName, cb_cluster)
	}
	bucket, err := cb_cluster.OpenBucket(BucketName, "")
	if err != nil {
		return nil, err
	}
	cb.create_indexes(bucket)
	c := &conn{
		db:     bucket,
		logger: logger,
	}
	return c, nil
}
