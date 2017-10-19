package blobstore_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"

	. "github.com/cloudfoundry-incubator/blobstore-backup-restore"
	"github.com/cloudfoundry-incubator/blobstore-backup-restore/fakes"
)

var _ = Describe("Backuper", func() {
	var dropletsBucket *fakes.FakeBucket
	var buildpacksBucket *fakes.FakeBucket
	var packagesBucket *fakes.FakeBucket

	var artifact *fakes.FakeArtifact

	var err error

	var backuper Backuper

	BeforeEach(func() {
		dropletsBucket = new(fakes.FakeBucket)
		buildpacksBucket = new(fakes.FakeBucket)
		packagesBucket = new(fakes.FakeBucket)

		artifact = new(fakes.FakeArtifact)

		backuper = NewBackuper([]Bucket{dropletsBucket, buildpacksBucket, packagesBucket}, artifact)
	})

	JustBeforeEach(func() {
		err = backuper.Backup()
	})

	Context("when the buckets have data", func() {
		BeforeEach(func() {
			dropletsBucket.IdentifierReturns("droplets")
			dropletsBucket.NameReturns("my_droplets_bucket")
			dropletsBucket.RegionNameReturns("my_droplets_region")
			dropletsBucket.VersionsReturns([]Version{
				{Key: "one", Id: "11", IsLatest: false},
				{Key: "one", Id: "12", IsLatest: false},
				{Key: "one", Id: "13", IsLatest: true},
				{Key: "two", Id: "21", IsLatest: false},
				{Key: "two", Id: "22", IsLatest: true},
			}, nil)

			buildpacksBucket.IdentifierReturns("buildpacks")
			buildpacksBucket.NameReturns("my_buildpacks_bucket")
			buildpacksBucket.RegionNameReturns("my_buildpacks_region")
			buildpacksBucket.VersionsReturns([]Version{
				{Key: "three", Id: "31", IsLatest: false},
				{Key: "three", Id: "32", IsLatest: true},
			}, nil)

			packagesBucket.IdentifierReturns("packages")
			packagesBucket.NameReturns("my_packages_bucket")
			packagesBucket.RegionNameReturns("my_packages_region")
			packagesBucket.VersionsReturns([]Version{
				{Key: "four", Id: "41", IsLatest: false},
				{Key: "four", Id: "43", IsLatest: true},
				{Key: "four", Id: "42", IsLatest: false},
			}, nil)
		})

		It("stores the latest versions in the artifact", func() {
			Expect(artifact.SaveArgsForCall(0)).To(Equal(map[string]BucketBackup{
				"droplets": {
					BucketName: "my_droplets_bucket",
					RegionName: "my_droplets_region",
					Versions: []LatestVersion{
						{BlobKey: "one", Id: "13"},
						{BlobKey: "two", Id: "22"},
					},
				},
				"buildpacks": {
					BucketName: "my_buildpacks_bucket",
					RegionName: "my_buildpacks_region",
					Versions: []LatestVersion{
						{BlobKey: "three", Id: "32"},
					},
				},
				"packages": {
					BucketName: "my_packages_bucket",
					RegionName: "my_packages_region",
					Versions: []LatestVersion{
						{BlobKey: "four", Id: "43"},
					},
				},
			}))
		})
	})

	Context("when retrieving the versions from the buckets fails", func() {
		BeforeEach(func() {
			dropletsBucket.VersionsReturns([]Version{}, nil)
			dropletsBucket.NameReturns("my_droplets_bucket")

			buildpacksBucket.VersionsReturns([]Version{}, errors.New("failed to retrieve versions"))
			buildpacksBucket.NameReturns("my_buildpacks_bucket")

			packagesBucket.VersionsReturns([]Version{}, nil)
			packagesBucket.NameReturns("my_packages_bucket")
		})

		It("returns the error from the bucket", func() {
			Expect(err).To(MatchError("failed to retrieve versions"))
		})
	})

	Context("when storing the versions in the artifact fails", func() {
		BeforeEach(func() {
			dropletsBucket.VersionsReturns([]Version{}, nil)
			dropletsBucket.NameReturns("my_droplets_bucket")

			buildpacksBucket.VersionsReturns([]Version{}, nil)
			buildpacksBucket.NameReturns("my_buildpacks_bucket")

			packagesBucket.VersionsReturns([]Version{}, nil)
			packagesBucket.NameReturns("my_packages_bucket")

			artifact.SaveReturns(errors.New("failed to save the versions artifact"))
		})

		It("returns the error from the artifact", func() {
			Expect(err).To(MatchError("failed to save the versions artifact"))
		})
	})
})