package postgresql

type Config struct {
	Host     string
	User     string
	Password string
	DbName   string
	Port     string
	SSLMode  string
	TimeZone string
}
