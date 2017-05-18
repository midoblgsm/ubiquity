package scbe_test

import (
	"encoding/json"
	"log"
	"net/http"

	"os"

	"github.com/IBM/ubiquity/local/scbe"
	"github.com/IBM/ubiquity/resources"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega" // including the whole package inside the file
	// httpmock is the referrer for this module
	//"gopkg.in/jarcoal/httpmock.v1"
	"github.com/jarcoal/httpmock"
)

const (
	fakeScbeQfdn        = "fake.com"
	fakeScbeUrlBase     = "https://" + fakeScbeQfdn + ":6666"
	suffix              = "api/v1"
	fakeScbeUrlAuth     = "users/get-auth-token"
	fakeScbeUrlAuthFull = fakeScbeUrlBase + "/" + suffix + "/" + fakeScbeUrlAuth
	fakeScbeUrlReferer  = fakeScbeUrlBase + "/"
)

var _ = Describe("restClient", func() {
	var (
		logger  *log.Logger
		conInfo resources.ConnectionInfo
		client  scbe.ScbeRestClient
		creds   resources.CredentialInfo
		err     error
	)
	BeforeEach(func() {
		logger = log.New(os.Stdout, "ubiquity scbe: ", log.Lshortfile|log.LstdFlags)
		creds = resources.CredentialInfo{UserName: "fake-user", Password: "fake-password"}
		conInfo = resources.ConnectionInfo{ManagementIP: "fake.com", Port: 6666}
		client, err = scbe.NewScbeRestClient(logger, conInfo, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	Context(".Login", func() {
		It("should succeed when httpClient succeed and return a token", func() {
			loginResponse := scbe.LoginResponse{Token: "fake-token"}
			marshalledResponse, err := json.Marshal(loginResponse)
			Expect(err).ToNot(HaveOccurred())

			httpmock.RegisterResponder(
				"POST",
				fakeScbeUrlAuthFull,
				httpmock.NewStringResponder(200, string(marshalledResponse)),
			)
			err = client.Login()
			Expect(err).ToNot(HaveOccurred())
		})
		It("should fail when httpClient succeed and return an empty token", func() {
			httpmock.RegisterResponder("POST", fakeScbeUrlAuthFull, httpmock.NewStringResponder(http.StatusOK, `{"token":""}`))
			err = client.Login()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Token is empty"))
		})
		It("should fail when httpClient fails to login due to bad status of response", func() {
			httpmock.RegisterResponder("POST", fakeScbeUrlAuthFull, httpmock.NewStringResponder(http.StatusBadRequest, "{}"))
			err = client.Login()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Error, bad status code of http response"))
		})
		It("should fail when httpClient.post return bad structure that marshaling cannot work with", func() {
			httpmock.RegisterResponder("POST", fakeScbeUrlAuthFull, httpmock.NewStringResponder(http.StatusOK, "yyy"))
			err = client.Login()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Error in unmarshalling response"))
		})

	})

})
