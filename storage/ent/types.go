package ent

// NetworkDB contains options common to SQL databases accessed over network.
type NetworkDB struct {
	Database string
	User     string
	Password string
	Host     string
	Port     uint16

	ConnectionTimeout int // Seconds

	MaxOpenConns    int // default: 5
	MaxIdleConns    int // default: 5
	ConnMaxLifetime int // Seconds, default: not set

	// RetryOnSerializationFailure enables bounded, jittered retry of refresh-token
	// updates aborted by transient serialization failures / deadlocks under
	// SERIALIZABLE isolation. Off by default. Retry parameters are not configurable.
	RetryOnSerializationFailure bool
}

// SSL represents SSL options for network databases.
type SSL struct {
	Mode   string
	CAFile string
	// Files for client auth.
	KeyFile  string
	CertFile string
}
