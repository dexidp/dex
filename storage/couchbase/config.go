package couchbase

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/couchbase/gocb.v1"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
)

const (
	quotaBucket  = 600
	dexIndexName = "idx_dex_dex_type_v01"
)

var BucketName string
var createBucketIfNotExists bool = false

// NetworkDB contains options to couchbase database accessed over network.
type NetworkDB struct {
	Bucket      string
	User        string
	Password    string
	Host        string
	Port        uint16
	CreateIndex bool
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

func (cb *Couchbase) create_index(bucket *gocb.Bucket) error {
	query := fmt.Sprintf("CREATE INDEX `%s` ON `%s`((meta().`id`)) "+
		"WHERE ((meta().`id`) like 'dex-%s') "+
		"WITH { 'defer_build':false }", dexIndexName, BucketName, "%")
	myQuery := gocb.NewN1qlQuery(query)
	_, err := bucket.ExecuteN1qlQuery(myQuery, nil)
	if err != nil {
		if !strings.Contains(err.Error(), fmt.Sprintf("The index %s already exists", dexIndexName)) {
			return err
		}
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
	if cb.CreateIndex {
		err = cb.create_index(bucket)
		if err != nil {
			return nil, err
		}
	}
	c := &conn{
		db:     bucket,
		logger: logger,
	}
	return c, nil
}
