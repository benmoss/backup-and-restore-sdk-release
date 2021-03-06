package postgres

import (
	"fmt"
	"log"

	"github.com/cloudfoundry-incubator/database-backup-restore/config"
	"github.com/cloudfoundry-incubator/database-backup-restore/runner"
	"github.com/cloudfoundry-incubator/database-backup-restore/version"
)

type ServerVersionDetector struct {
	psqlPath string
}

func NewServerVersionDetector(psqlPath string) ServerVersionDetector {
	return ServerVersionDetector{psqlPath: psqlPath}
}

func (d ServerVersionDetector) GetVersion(config config.ConnectionConfig) (version.SemanticVersion, error) {
	stdout, stderr, err := runner.Run(d.psqlPath, []string{"--tuples-only",
		fmt.Sprintf("--username=%s", config.Username),
		fmt.Sprintf("--host=%s", config.Host),
		fmt.Sprintf("--port=%d", config.Port),
		config.Database,
		`--command=SELECT VERSION()`},
		map[string]string{"PGPASSWORD": config.Password})

	if err != nil {
		log.Fatalf("Unable to check version of Postgres: %v\n%s\n%s", err, string(stdout), string(stderr))
	}

	return ParseVersion(string(stdout))
}
