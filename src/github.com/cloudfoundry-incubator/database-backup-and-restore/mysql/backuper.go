package mysql

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"

	"github.com/cloudfoundry-incubator/database-backup-and-restore/config"
	"github.com/cloudfoundry-incubator/database-backup-and-restore/runner"
	"github.com/cloudfoundry-incubator/database-backup-and-restore/version"
)

type Backuper struct {
	config       config.ConnectionConfig
	backupBinary string
	clientBinary string
}

func NewBackuper(config config.ConnectionConfig, utilitiesConfig config.UtilitiesConfig) Backuper {
	return Backuper{
		config:       config,
		backupBinary: utilitiesConfig.Mysql.Dump,
		clientBinary: utilitiesConfig.Mysql.Client,
	}
}

func (b Backuper) Action(artifactFilePath string) error {
	// sample output: "mysqldump  Ver 10.16 Distrib 10.1.22-MariaDB, for Linux (x86_64)"
	// /mysqldump\s+Ver\s+[^ ]+\s+Distrib\s+([^ ]+),/
	mysqldumpCmd := exec.Command(b.backupBinary, "-V")
	mysqldumpVersion := extractVersionUsingCommand(
		mysqldumpCmd,
		`^mysqldump\s+Ver\s+[^ ]+\s+Distrib\s+([^ ]+),`)

	log.Printf("%s version %v\n", b.backupBinary, mysqldumpVersion)

	// extract version from mysql server
	mysqlClientCmd := exec.Command(b.clientBinary,
		"--skip-column-names",
		"--silent",
		fmt.Sprintf("--user=%s", b.config.Username),
		fmt.Sprintf("--password=%s", b.config.Password),
		fmt.Sprintf("--host=%s", b.config.Host),
		fmt.Sprintf("--port=%d", b.config.Port),
		"--execute=SELECT VERSION()")
	mysqlServerVersion := extractVersionUsingCommand(mysqlClientCmd, `(.+)`)

	log.Printf("MYSQL server (%s:%d) version %v\n", b.config.Host, b.config.Port, mysqlServerVersion)

	// compare versions: for ServerX.ServerY.ServerZ and DumpX.DumpY.DumpZ
	// 	=> ServerX != DumpX => error
	//	=> ServerY != DumpY => error
	// ServerZ and DumpZ are regarded as patch version and compatibility is assumed
	if !mysqlServerVersion.MinorVersionMatches( mysqldumpVersion){
		log.Fatalf("Version mismatch between mysqldump %s and the MYSQL server %s\n"+
			"mysqldump utility and the MYSQL server must be at the same major and minor version.\n",
			mysqldumpVersion,
			mysqlServerVersion)
	}

	cmdArgs := []string{
		"-v",
		"--single-transaction",
		"--skip-add-locks",
		"--user=" + b.config.Username,
		"--host=" + b.config.Host,
		fmt.Sprintf("--port=%d", b.config.Port),
		"--result-file=" + artifactFilePath,
		b.config.Database,
	}

	cmdArgs = append(cmdArgs, b.config.Tables...)

	_, _, err := runner.Run(b.backupBinary, cmdArgs, map[string]string{"MYSQL_PWD": b.config.Password})

	return err
}

func extractVersionUsingCommand(cmd *exec.Cmd, searchPattern string) version.SemanticVersion {
	stdout, err := cmd.Output()
	if err != nil {
		log.Fatalln("Error running command.", err)
	}

	r := regexp.MustCompile(searchPattern)
	matches := r.FindSubmatch(stdout)
	if matches == nil {
		log.Fatalln("Could not determine version by using search pattern:", searchPattern)
	}

	versionString := matches[1]

	r = regexp.MustCompile(`(\d+).(\d+).(\S+)`)
	matches = r.FindSubmatch(versionString)
	if matches == nil {
		log.Fatalln("Could not determine version by using search pattern:", searchPattern)
	}

	return version.SemanticVersion{
		Major: string(matches[1]),
		Minor: string(matches[2]),
		Patch: string(matches[3]),
	}
}