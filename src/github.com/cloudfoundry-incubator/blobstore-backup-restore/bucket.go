package blobstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

//go:generate counterfeiter -o fakes/fake_bucket.go . Bucket
type Bucket interface {
	Name() string
	RegionName() string
	Versions() ([]Version, error)
	PutVersions(regionName, bucketName string, versions []LatestVersion) error
}

type S3Bucket struct {
	awsCliPath string
	name       string
	regionName string
	accessKey  S3AccessKey
}

type S3AccessKey struct {
	Id     string
	Secret string
}

func NewS3Bucket(awsCliPath, name, region string, accessKey S3AccessKey) S3Bucket {
	return S3Bucket{
		awsCliPath: awsCliPath,
		name:       name,
		regionName: region,
		accessKey:  accessKey,
	}
}

func (b S3Bucket) Name() string {
	return b.name
}

func (b S3Bucket) RegionName() string {
	return b.regionName
}

func (b S3Bucket) Versions() ([]Version, error) {
	output, err := b.runS3ApiCommand("list-object-versions", "--bucket", b.name)
	if err != nil {
		return nil, err
	}

	response := S3ListVersionsResponse{}
	err = json.Unmarshal(output, &response)
	if err != nil {
		return nil, err
	}

	return response.Versions, nil
}

func (b S3Bucket) PutVersions(regionName, bucketName string, versions []LatestVersion) error {
	var err error

	for _, version := range versions {
		err = b.putVersion(regionName, bucketName, version)
		if err != nil {
			return err
		}
	}

	files, err := b.listFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		included := versionsIncludeFile(file, versions)
		if !included {
			err = b.deleteFile(file)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (b S3Bucket) putVersion(regionName, bucketName string, version LatestVersion) error {
	_, err := b.runS3ApiCommand(
		"copy-object",
		"--bucket", b.name,
		"--key", version.BlobKey,
		"--copy-source", fmt.Sprintf("/%s/%s?versionId=%s", bucketName, version.BlobKey, version.Id),
	)

	return err
}

func (b S3Bucket) listFiles() ([]string, error) {
	output, err := b.runS3ApiCommand("list-objects", "--bucket", b.name)
	if err != nil {
		return nil, err
	}

	response := S3ListResponse{}
	err = json.Unmarshal(output, &response)
	if err != nil {
		return nil, err
	}

	files := []string{}
	for _, object := range response.Contents {
		files = append(files, object.Key)
	}

	return files, nil
}

func (b S3Bucket) deleteFile(key string) error {
	_, err := b.runS3ApiCommand(
		"delete-object",
		"--bucket", b.name,
		"--key", key,
	)

	return err
}

func (b S3Bucket) runS3ApiCommand(args ...string) ([]byte, error) {
	outputBuffer := new(bytes.Buffer)
	errorBuffer := new(bytes.Buffer)

	baseArgs := []string{"--output", "json", "--region", b.regionName, "s3api"}
	awsCmd := exec.Command(b.awsCliPath, append(baseArgs, args...)...)
	awsCmd.Env = append(awsCmd.Env, "AWS_ACCESS_KEY_ID="+b.accessKey.Id)
	awsCmd.Env = append(awsCmd.Env, "AWS_SECRET_ACCESS_KEY="+b.accessKey.Secret)
	awsCmd.Stdout = outputBuffer
	awsCmd.Stderr = errorBuffer

	err := awsCmd.Run()
	if err != nil {
		return nil, errors.New(errorBuffer.String())
	}

	return outputBuffer.Bytes(), nil
}

type S3ListVersionsResponse struct {
	Versions []Version
}

type Version struct {
	Key      string
	Id       string `json:"VersionId"`
	IsLatest bool
}

type S3ListResponse struct {
	Contents []Object
}

type Object struct {
	Key string
}

func versionsIncludeFile(file string, versions []LatestVersion) bool {
	for _, version := range versions {
		if version.BlobKey == file {
			return true
		}
	}

	return false
}
