package scbe_test

import (
	"fmt"
	"log"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/IBM/ubiquity/fakes"
	"github.com/IBM/ubiquity/resources"

	"github.com/IBM/ubiquity/local/scbe"
)

var _ = Describe("scbeLocalClient", func() {
	var (
		client             resources.StorageClient
		logger             *log.Logger
		fakeScbeDataModel  *fakes.FakeScbeDataModel
		fakeScbeRestClient *fakes.FakeScbeRestClient
		fakeConfig         resources.ScbeConfig
		err                error
	)
	BeforeEach(func() {
		logger = log.New(os.Stdout, "ubiquity scbe: ", log.Lshortfile|log.LstdFlags)
		fakeScbeDataModel = new(fakes.FakeScbeDataModel)
		fakeScbeRestClient = new(fakes.FakeScbeRestClient)
		fakeConfig = resources.ScbeConfig{ConfigPath: "/tmp"} // TODO add more details
		client, err = scbe.NewScbeLocalClientWithNewScbeRestClientAndDataModel(
			logger,
			fakeConfig,
			fakeScbeDataModel,
			fakeScbeRestClient)
		Expect(err).ToNot(HaveOccurred())

	})

	Context(".Activate", func() {
		It("should fail when scbeRestClient returns error", func() {
			fakeScbeRestClient.LoginReturns(fmt.Errorf("Error to login"))
			err = client.Activate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Error in login remote call"))
			Expect(fakeScbeRestClient.LoginCallCount()).To(Equal(1))
			Expect(fakeScbeRestClient.ServiceExistCallCount()).To(Equal(0))

		})

		It("should fail when ServiceExist returns error", func() {
			fakeScbeRestClient.LoginReturns(nil)
			fakeScbeRestClient.ServiceExistReturns(true, fmt.Errorf("fake-error"))
			err = client.Activate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Error in serviceExist remote call"))
			Expect(fakeScbeRestClient.LoginCallCount()).To(Equal(1))
			Expect(fakeScbeRestClient.ServiceExistCallCount()).To(Equal(1))

		})
		It("should fail when ServiceExist returns false", func() {
			fakeScbeRestClient.LoginReturns(nil)
			fakeScbeRestClient.ServiceExistReturns(false, nil)
			err = client.Activate()
			Expect(err).To(HaveOccurred())
			Expect(fakeScbeRestClient.LoginCallCount()).To(Equal(1))
			Expect(fakeScbeRestClient.ServiceExistCallCount()).To(Equal(1))

		})
		It("should succeed when ServiceExist returns true", func() {
			fakeScbeRestClient.LoginReturns(nil)
			fakeScbeRestClient.ServiceExistReturns(true, nil)
			err = client.Activate()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeScbeRestClient.LoginCallCount()).To(Equal(1))
			Expect(fakeScbeRestClient.ServiceExistCallCount()).To(Equal(1))

		})

	})

})
