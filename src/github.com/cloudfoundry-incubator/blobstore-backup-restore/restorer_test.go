package blobstore_test

import (
	. "github.com/cloudfoundry-incubator/blobstore-backup-restore"

	"errors"

	"github.com/cloudfoundry-incubator/blobstore-backup-restore/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Restorer", func() {
	var dropletsBucket *fakes.FakeBucket
	var buildpacksBucket *fakes.FakeBucket
	var packagesBucket *fakes.FakeBucket

	var artifact *fakes.FakeArtifact

	var err error

	var restorer Restorer

	BeforeEach(func() {
		dropletsBucket = new(fakes.FakeBucket)
		buildpacksBucket = new(fakes.FakeBucket)
		packagesBucket = new(fakes.FakeBucket)

		artifact = new(fakes.FakeArtifact)

		restorer = NewRestorer(map[string]Bucket{
			"droplets":   dropletsBucket,
			"buildpacks": buildpacksBucket,
			"packages":   packagesBucket,
		}, artifact)
	})

	JustBeforeEach(func() {
		err = restorer.Restore()
	})

	Context("when the artifact is valid and copying versions to buckets works", func() {
		BeforeEach(func() {
			artifact.LoadReturns(map[string]BucketBackup{
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
			}, nil)

			dropletsBucket.PutVersionsReturns(nil)
			buildpacksBucket.PutVersionsReturns(nil)
			packagesBucket.PutVersionsReturns(nil)
		})

		It("restores a backup to the corresponding buckets", func() {
			Expect(err).NotTo(HaveOccurred())

			expectedSourceRegionName, expectedSourceBucketName, expectedVersions := dropletsBucket.PutVersionsArgsForCall(0)
			Expect(expectedSourceBucketName).To(Equal("my_droplets_bucket"))
			Expect(expectedSourceRegionName).To(Equal("my_droplets_region"))
			Expect(expectedVersions).To(Equal([]LatestVersion{
				{BlobKey: "one", Id: "13"},
				{BlobKey: "two", Id: "22"},
			}))

			expectedSourceRegionName, expectedSourceBucketName, expectedVersions = buildpacksBucket.PutVersionsArgsForCall(0)
			Expect(expectedSourceBucketName).To(Equal("my_buildpacks_bucket"))
			Expect(expectedSourceRegionName).To(Equal("my_buildpacks_region"))
			Expect(expectedVersions).To(Equal([]LatestVersion{
				{BlobKey: "three", Id: "32"},
			}))

			expectedSourceRegionName, expectedSourceBucketName, expectedVersions = packagesBucket.PutVersionsArgsForCall(0)
			Expect(expectedSourceBucketName).To(Equal("my_packages_bucket"))
			Expect(expectedSourceRegionName).To(Equal("my_packages_region"))
			Expect(expectedVersions).To(Equal([]LatestVersion{
				{BlobKey: "four", Id: "43"},
			}))
		})
	})

	Context("when the artifact fails to load", func() {
		BeforeEach(func() {
			artifact.LoadReturns(nil, errors.New("artifact failed to load"))
		})

		It("stops and returns an error", func() {
			Expect(err).To(MatchError("artifact failed to load"))
			Expect(dropletsBucket.PutVersionsCallCount()).To(Equal(0))
			Expect(buildpacksBucket.PutVersionsCallCount()).To(Equal(0))
			Expect(packagesBucket.PutVersionsCallCount()).To(Equal(0))
		})
	})

	Context("when copying versions on a bucket fails", func() {
		BeforeEach(func() {
			artifact.LoadReturns(map[string]BucketBackup{
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
			}, nil)

			dropletsBucket.PutVersionsReturns(nil)
			buildpacksBucket.PutVersionsReturns(errors.New("failed to put versions to bucket 'buildpacks'"))
			packagesBucket.PutVersionsReturns(nil)
		})

		It("stops and returns an error", func() {
			Expect(err).To(MatchError("failed to put versions to bucket 'buildpacks'"))

			expectedSourceRegionName, expectedSourceBucketName, expectedVersions := buildpacksBucket.PutVersionsArgsForCall(0)
			Expect(expectedSourceBucketName).To(Equal("my_buildpacks_bucket"))
			Expect(expectedSourceRegionName).To(Equal("my_buildpacks_region"))
			Expect(expectedVersions).To(Equal([]LatestVersion{
				{BlobKey: "three", Id: "32"},
			}))
		})
	})
})
