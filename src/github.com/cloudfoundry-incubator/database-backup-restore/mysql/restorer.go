package mysql

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/cloudfoundry-incubator/database-backup-restore/config"
)

type Restorer struct {
	config       config.ConnectionConfig
	clientBinary string
}

func NewRestorer(config config.ConnectionConfig, restoreBinary string) Restorer {
	return Restorer{
		config:       config,
		clientBinary: restoreBinary,
	}
}

func (r Restorer) Action(artifactFilePath string) error {
	artifactFile, err := os.Open(artifactFilePath)
	if err != nil {
		log.Fatalln("Error reading from artifact file,", err)
	}

	cmd := exec.Command(r.clientBinary,
		"-v",
		"--user="+r.config.Username,
		"--host="+r.config.Host,
		fmt.Sprintf("--port=%d", r.config.Port),
		r.config.Database,
	)

	cmd.Stdin = bufio.NewReader(artifactFile)
	cmd.Env = append(cmd.Env, "MYSQL_PWD="+r.config.Password)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
