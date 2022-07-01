package db

type DBConfig struct {
	Host         string
	Port         int
	Name         string
	Username     string
	Password     string
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	ShowSQL      bool   `yaml:"show_sql"`
	Loc          string `yaml:"loc"`
}
