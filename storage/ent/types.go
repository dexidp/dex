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
}

// SSL represents SSL options for network databases.
type SSL struct {
	Mode   string
	CAFile string
	// Files for client auth.
	KeyFile  string
	CertFile string
}
