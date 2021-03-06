package main

import (
	"log"
	"os"

	"encoding/json"
	"io/ioutil"

	"errors"
	"flag"

	"github.com/cloudfoundry-incubator/blobstore-backup-restore"
)

func main() {
	commandFlags, err := parseFlags()
	if err != nil {
		log.Fatal(err.Error())
	}

	awsCliPath := getEnv("AWS_CLI_PATH")

	artifact := blobstore.NewFileArtifact(commandFlags.ArtifactFilePath)

	config, err := ioutil.ReadFile(commandFlags.ConfigPath)
	if err != nil {
		log.Fatal("Failed to read config")
	}

	var bucketsConfig map[string]BucketConfig
	err = json.Unmarshal(config, &bucketsConfig)
	if err != nil {
		log.Fatal("Failed to parse config")
	}

	buckets := makeBuckets(awsCliPath, bucketsConfig)

	if commandFlags.IsRestore {
		err = blobstore.NewRestorer(buckets, artifact).Restore()
	} else {
		err = blobstore.NewBackuper(buckets, artifact).Backup()
	}

	if err != nil {
		log.Fatal(err.Error())
	}
}

func getEnv(varName string) string {
	value, exists := os.LookupEnv(varName)
	if !exists {
		log.Fatalf("Missing environment variable '%s'", varName)
	}
	return value
}

func makeBuckets(awsCliPath string, config map[string]BucketConfig) map[string]blobstore.Bucket {
	var buckets = map[string]blobstore.Bucket{}

	for identifier, bucketConfig := range config {
		buckets[identifier] = blobstore.NewS3Bucket(
			awsCliPath,
			bucketConfig.Name,
			bucketConfig.Region,
			blobstore.S3AccessKey{
				Id:     bucketConfig.AwsAccessKeyId,
				Secret: bucketConfig.AwsSecretAccessKey,
			},
		)
	}

	return buckets
}

type BucketConfig struct {
	Name               string `json:"name"`
	Region             string `json:"region"`
	AwsAccessKeyId     string `json:"aws_access_key_id"`
	AwsSecretAccessKey string `json:"aws_secret_access_key"`
}

func parseFlags() (CommandFlags, error) {
	var configFilePath = flag.String("config", "", "Path to JSON config file")
	var backupAction = flag.Bool("backup", false, "Run blobstore backup")
	var restoreAction = flag.Bool("restore", false, "Run blobstore restore")
	var artifactFilePath = flag.String("artifact-file", "", "Path to the artifact file")

	flag.Parse()

	if *backupAction && *restoreAction {
		return CommandFlags{}, errors.New("only one of: --backup or --restore can be provided")
	}

	if !*backupAction && !*restoreAction {
		return CommandFlags{}, errors.New("missing --backup or --restore flag")
	}

	if *configFilePath == "" {
		return CommandFlags{}, errors.New("missing --config flag")
	}

	if *artifactFilePath == "" {
		return CommandFlags{}, errors.New("missing --artifact-file flag")
	}

	return CommandFlags{
		ConfigPath:       *configFilePath,
		IsRestore:        *restoreAction,
		ArtifactFilePath: *artifactFilePath,
	}, nil
}

type CommandFlags struct {
	ConfigPath       string
	IsRestore        bool
	ArtifactFilePath string
}
