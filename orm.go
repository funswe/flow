package flow

type Config struct {
	userName string
	password string
	dbName   string
	host     string
	port     uint
	pool     *Pool
}

type Pool struct {
	maxIdle         int
	maxOpen         int
	connMaxLifetime int
}

func NewConfig(userName, password, dbName, host string, port uint, pool *Pool) *Config {
	return &Config{userName, password, dbName, host, port, pool}
}

func NewPool(maxIdle, maxOpen, connMaxLifetime int) *Pool {
	return &Pool{maxIdle, maxOpen, connMaxLifetime}
}
