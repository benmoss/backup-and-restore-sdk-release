package config

import (
	"log"
	"os"
)

type UtilityPaths struct {
	Client  string
	Dump    string
	Restore string
}

type UtilitiesConfig struct {
	Postgres96 UtilityPaths
	Postgres94 UtilityPaths
	Mysql      UtilityPaths
}

func GetUtilitiesConfigFromEnv() UtilitiesConfig {
	return UtilitiesConfig{
		Postgres96: UtilityPaths{
			Client:  lookupEnv("PG_CLIENT_PATH"),
			Dump:    lookupEnv("PG_DUMP_9_6_PATH"),
			Restore: lookupEnv("PG_RESTORE_9_6_PATH"),
		},
		Postgres94: UtilityPaths{
			Client:  lookupEnv("PG_CLIENT_PATH"),
			Dump:    lookupEnv("PG_DUMP_9_4_PATH"),
			Restore: lookupEnv("PG_RESTORE_9_4_PATH"),
		},
		Mysql: UtilityPaths{
			Client:  lookupEnv("MYSQL_CLIENT_PATH"),
			Dump:    lookupEnv("MYSQL_DUMP_PATH"),
			Restore: lookupEnv("MYSQL_CLIENT_PATH"),
		},
	}
}

func lookupEnv(key string) string {
	value, valueSet := os.LookupEnv(key)
	if !valueSet {
		log.Fatalln(key + " must be set")
	}
	return value
}
