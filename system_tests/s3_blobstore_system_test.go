package system_tests

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"time"

	"strconv"

	"fmt"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("S3 backuper", func() {
	var region string
	var cloneRegion string
	var bucket string
	var cloneBucket string
	var fileName1 string
	var fileName2 string
	var fileName3 string
	var artifactDirPath string

	var backuperInstance JobInstance
	var cloneBackuperInstance JobInstance

	BeforeEach(func() {
		backuperInstance = JobInstance{
			deployment:    MustHaveEnv("BOSH_DEPLOYMENT"),
			instance:      "backuper",
			instanceIndex: "0",
		}
		cloneBackuperInstance = JobInstance{
			deployment:    MustHaveEnv("BOSH_DEPLOYMENT"),
			instance:      "clone-backuper",
			instanceIndex: "0",
		}

		region = MustHaveEnv("AWS_TEST_BUCKET_REGION")
		bucket = MustHaveEnv("AWS_TEST_BUCKET_NAME")
		cloneRegion = MustHaveEnv("AWS_TEST_CLONE_BUCKET_REGION")
		cloneBucket = MustHaveEnv("AWS_TEST_CLONE_BUCKET_NAME")

		artifactDirPath = "/tmp/aws-s3-versioned-blobstore-backup-restorer" + strconv.FormatInt(time.Now().Unix(), 10)
		backuperInstance.runOnVMAndSucceed("mkdir -p " + artifactDirPath)
		cloneBackuperInstance.runOnVMAndSucceed("mkdir -p " + artifactDirPath)
	})

	AfterEach(func() {
		deleteAllVersionsFromBucket(region, bucket)
		deleteAllVersionsFromBucket(cloneRegion, cloneBucket)
		backuperInstance.runOnVMAndSucceed("rm -rf " + artifactDirPath)
		cloneBackuperInstance.runOnVMAndSucceed("rm -rf " + artifactDirPath)
	})

	It("backs up and restores in-place", func() {
		fileName1 = uploadTimestampedFileToBucket(region, bucket, "file1", "FILE1")
		fileName2 = uploadTimestampedFileToBucket(region, bucket, "file2", "FILE2")

		backuperInstance.runOnVMAndSucceed("BBR_ARTIFACT_DIRECTORY=" + artifactDirPath +
			" /var/vcap/jobs/aws-s3-versioned-blobstore-backup-restorer/bin/bbr/backup")

		deleteFileFromBucket(region, bucket, fileName1)
		writeFileInBucket(region, bucket, fileName2, "FILE2_NEW")
		fileName3 = uploadTimestampedFileToBucket(region, bucket, "file3", "FILE3")

		backuperInstance.runOnVMAndSucceed("BBR_ARTIFACT_DIRECTORY=" + artifactDirPath +
			" /var/vcap/jobs/aws-s3-versioned-blobstore-backup-restorer/bin/bbr/restore")

		filesList := listFilesFromBucket(region, bucket)
		Expect(filesList).To(ConsistOf(fileName1, fileName2))

		Expect(getFileContentsFromBucket(region, bucket, fileName1)).To(Equal("FILE1"))
		Expect(getFileContentsFromBucket(region, bucket, fileName2)).To(Equal("FILE2"))
	})

	It("backs up and restores to a cloned bucket", func() {
		fileName1 = uploadTimestampedFileToBucket(region, bucket, "file1", "FILE1")
		fileName2 = uploadTimestampedFileToBucket(region, bucket, "file2", "FILE2")

		backuperInstance.runOnVMAndSucceed("BBR_ARTIFACT_DIRECTORY=" + artifactDirPath +
			" /var/vcap/jobs/aws-s3-versioned-blobstore-backup-restorer/bin/bbr/backup")

		backuperInstance.downloadFromInstance(artifactDirPath+"/blobstore.json", "/tmp/blobstore.json")
		cloneBackuperInstance.uploadToInstance("/tmp/blobstore.json", artifactDirPath+"/blobstore.json")

		cloneBackuperInstance.runOnVMAndSucceed("BBR_ARTIFACT_DIRECTORY=" + artifactDirPath +
			" /var/vcap/jobs/aws-s3-versioned-blobstore-backup-restorer/bin/bbr/restore")

		filesList := listFilesFromBucket(cloneRegion, cloneBucket)
		Expect(filesList).To(ConsistOf(fileName1, fileName2))

		Expect(getFileContentsFromBucket(cloneRegion, cloneBucket, fileName1)).To(Equal("FILE1"))
		Expect(getFileContentsFromBucket(cloneRegion, cloneBucket, fileName2)).To(Equal("FILE2"))
	})
})

func getFileContentsFromBucket(region, bucket, key string) string {
	outputBuffer := runAwsCommandOnBucket(
		"--region", region,
		"s3",
		"cp",
		fmt.Sprintf("s3://%s/%s", bucket, key),
		"-")

	return outputBuffer.String()
}

func listFilesFromBucket(region, bucket string) []string {
	outputBuffer := runAwsCommandOnBucket(
		"--region", region,
		"s3api",
		"list-objects",
		"--bucket", bucket)

	var response ListResponse
	json.Unmarshal(outputBuffer.Bytes(), &response)

	keys := []string{}
	for _, entry := range response.Contents {
		keys = append(keys, entry.Key)
	}

	return keys
}

func uploadTimestampedFileToBucket(region, bucket, prefix, body string) string {
	fileName := prefix + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	writeFileInBucket(region, bucket, fileName, body)
	return fileName
}

func writeFileInBucket(region, bucket, key, body string) string {
	bodyFile, _ := ioutil.TempFile("", key)
	bodyFile.WriteString(body)
	bodyFile.Close()

	outputBuffer := runAwsCommandOnBucket(
		"--region", region,
		"s3api",
		"put-object",
		"--bucket", bucket,
		"--key", key,
		"--body", bodyFile.Name())

	var response PutResponse
	json.Unmarshal(outputBuffer.Bytes(), &response)

	Expect(response.VersionId).NotTo(BeEmpty(), "Bucket must be versioned!")

	return response.VersionId
}

func deleteFileFromBucket(region, bucket, key string) string {
	outputBuffer := runAwsCommandOnBucket(
		"--region", region,
		"s3api",
		"delete-object",
		"--bucket", bucket,
		"--key", key)

	var response PutResponse
	json.Unmarshal(outputBuffer.Bytes(), &response)

	return response.VersionId
}

func deleteVersionFromBucket(region, bucket, key, versionId string) string {
	outputBuffer := runAwsCommandOnBucket(
		"--region", region,
		"s3api",
		"delete-object",
		"--bucket", bucket,
		"--key", key,
		"--version-id", versionId)

	var response PutResponse
	json.Unmarshal(outputBuffer.Bytes(), &response)

	return response.VersionId
}

func deleteAllVersionsFromBucket(region, bucket string) {
	outputBuffer := runAwsCommandOnBucket(
		"--region", region,
		"s3api",
		"list-object-versions",
		"--bucket", bucket)

	var response VersionsResponse
	json.Unmarshal(outputBuffer.Bytes(), &response)

	for _, version := range response.Versions {
		deleteVersionFromBucket(region, bucket, version.Key, version.VersionId)
	}

	for _, version := range response.DeleteMarkers {
		deleteVersionFromBucket(region, bucket, version.Key, version.VersionId)
	}
}

func runAwsCommandOnBucket(args ...string) *bytes.Buffer {
	outputBuffer := new(bytes.Buffer)
	errorBuffer := new(bytes.Buffer)

	awsCmd := exec.Command("aws", args...)
	awsCmd.Stdout = outputBuffer
	awsCmd.Stderr = errorBuffer

	err := awsCmd.Run()
	Expect(err).ToNot(HaveOccurred(), errorBuffer.String())

	return outputBuffer
}

type PutResponse struct {
	VersionId string
}

type ListResponse struct {
	Contents []ListResponseEntry
}

type ListResponseEntry struct {
	Key string
}

type VersionsResponse struct {
	Versions      []VersionItem
	DeleteMarkers []VersionItem
}

type VersionItem struct {
	Key       string
	VersionId string
}
