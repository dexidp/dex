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

// BucketName the name of the couchbase bucket is going to be used for saving dex data
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

// SSL configurations is not fully tested
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

func (cb *Couchbase) createBucket(logger log.Logger, bucketName string, cbCluster *gocb.Cluster) error {
	clusterManager := cbCluster.Manager(cb.User, cb.Password)
	bucketSettings := gocb.BucketSettings{
		FlushEnabled:  false,
		IndexReplicas: false,
		Name:          bucketName,
		Password:      "",
		Quota:         quotaBucket,
		Replicas:      1,
	}
	err := clusterManager.InsertBucket(&bucketSettings)
	if err != nil {
		logger.Errorf(err.Error())
		return err
	}
	time.Sleep(3 * time.Second)
	return nil
}

func (cb *Couchbase) bucketExists(cbCluster *gocb.Cluster) bool {
	clusterManager := cbCluster.Manager(cb.User, cb.Password)
	listBuckets, err := clusterManager.GetBuckets()
	exists := false
	if err == nil {
		for _, bucketVar := range listBuckets {
			if bucketVar.Name != "" && bucketVar.Name == BucketName {
				exists = true
				break
			}
		}

	}
	return exists
}

func (cb *Couchbase) createIndex(bucket *gocb.Bucket) error {
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

func (cb *Couchbase) getConnectionString() string {
	connectionString := fmt.Sprintf("couchbase://%s", dataSourceStr(cb.Host))
	if cb.SSL.CertFile != "" {
		connectionString = fmt.Sprintf("couchbases://%s?certpath=%s", dataSourceStr(cb.Host), cb.SSL.CertFile)
	}
	return connectionString
}

func (cb *Couchbase) open(logger log.Logger) (*conn, error) {
	BucketName = cb.Bucket
	connectionString := cb.getConnectionString()
	cbCluster, err := gocb.Connect(connectionString)
	if err != nil {
		return nil, err
	}
	cbCluster.Authenticate(gocb.PasswordAuthenticator{
		Username: cb.User,
		Password: cb.Password,
	})
	if createBucketIfNotExists && !cb.bucketExists(cbCluster) {
		cb.createBucket(logger, BucketName, cbCluster)
	}
	bucket, err := cbCluster.OpenBucket(BucketName, "")
	if err != nil {
		return nil, err
	}
	if cb.CreateIndex {
		err = cb.createIndex(bucket)
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
