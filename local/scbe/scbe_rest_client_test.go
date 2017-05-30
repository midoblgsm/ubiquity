package scbe_test

import (
	"encoding/json"
	"fmt"
	"github.com/IBM/ubiquity/fakes"
	"github.com/IBM/ubiquity/local/scbe"
	"github.com/IBM/ubiquity/resources"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega" // including the whole package inside the file
	"log"
	"net/http"
	"os"
)

const (
	fakeScbeQfdn        = "1.1.1.1"
	fakeScbeUrlBase     = "https://" + fakeScbeQfdn + ":6666"
	suffix              = "api/v1"
	fakeScbeUrlAuth     = "users/get-auth-token"
	fakeScbeUrlAuthFull = fakeScbeUrlBase + "/" + suffix + "/" + fakeScbeUrlAuth
	fakeScbeUrlReferer  = fakeScbeUrlBase + "/"
	fakeScbeUrlApi      = fakeScbeUrlBase + "/" + suffix
	fakeProfileName     = "fake_profile"
)

var fakeServiceJsonResponse string = `
[
{
"id": "cc4c1254-d551-4a51-81f5-ffffffffffff",
"unique_identifier": "cc4c1254-d551-4a51-81f5-ffffffffffff",
"name": "gold",
"description": " ",
"container": "23c380fc-fe1e-4c02-9d1e-ffffffffffff",
"capability_values": "",
"type": "regular",
"physical_size": 413457711104,
"logical_size": 413457711104,
"physical_free": 310093283328,
"logical_free": 310093283328,
"total_capacity": 413457711104,
"used_capacity": 103364427776,
"max_resource_logical_free": 310093283328,
"max_resource_free_size_for_provisioning": 310093283328,
"num_volumes": 0,
"has_admin": true,
"qos_max_iops": 0,
"qos_max_mbps": 0
}
]`

var _ = Describe("restClient", func() {
	var (
		logger *log.Logger
		client scbe.RestClient
		err    error
	)
	BeforeEach(func() {
		logger = log.New(os.Stdout, "ubiquity scbe: ", log.Lshortfile|log.LstdFlags)
		client = scbe.NewRestClient(logger, resources.ConnectionInfo{}, fakeScbeUrlBase+"/"+suffix, fakeScbeUrlAuth, fakeScbeUrlReferer)
	})

	Context(".Login", func() {
		It("should succeed when httpClient succeed and return a token", func() {
			loginResponse := scbe.LoginResponse{Token: "fake-token"}
			marshalledResponse, err := json.Marshal(loginResponse)
			Expect(err).ToNot(HaveOccurred())
			httpmock.RegisterResponder(
				"POST",
				fakeScbeUrlAuthFull,
				httpmock.NewStringResponder(http.StatusOK, string(marshalledResponse)),
			)
			err = client.Login()
			Expect(err).ToNot(HaveOccurred())
		})
		It("should fail when httpClient succeed and return an empty token", func() {
			loginResponse := scbe.LoginResponse{Token: ""}
			marshalledResponse, err := json.Marshal(loginResponse)
			Expect(err).ToNot(HaveOccurred())
			httpmock.RegisterResponder("POST", fakeScbeUrlAuthFull, httpmock.NewStringResponder(http.StatusOK, string(marshalledResponse)))
			err = client.Login()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Token is empty"))
		})
		It("should fail when httpClient fails to login due to bad status of response", func() {
			httpmock.RegisterResponder("POST", fakeScbeUrlAuthFull, httpmock.NewStringResponder(http.StatusBadRequest, "{}"))
			err = client.Login()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(MatchRegexp("^Error, bad status code"))
		})
		It("should fail when httpClient.post return bad structure that marshaling cannot work with", func() {
			httpmock.RegisterResponder("POST", fakeScbeUrlAuthFull, httpmock.NewStringResponder(http.StatusOK, "yyy"))
			err = client.Login()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(MatchRegexp("^invalid character"))
		})

	})
})

var _ = Describe("restClient", func() {
	var (
		logger *log.Logger
		client scbe.RestClient
		err    error
	)
	BeforeEach(func() {
		logger = log.New(os.Stdout, "ubiquity scbe: ", log.Lshortfile|log.LstdFlags)
		client = scbe.NewRestClient(logger, resources.ConnectionInfo{}, fakeScbeUrlBase+"/"+suffix, fakeScbeUrlAuth, fakeScbeUrlReferer)
		loginResponse := scbe.LoginResponse{Token: "fake-token"}
		marshalledResponse, err := json.Marshal(loginResponse)
		Expect(err).ToNot(HaveOccurred())
		httpmock.RegisterResponder(
			"POST",
			fakeScbeUrlAuthFull,
			httpmock.NewStringResponder(http.StatusOK, string(marshalledResponse)),
		)
		err = client.Login()
		Expect(err).ToNot(HaveOccurred())

	})

	Context(".Get", func() {
		It("should succeed when Get succeed and return an expacted struct back", func() {
			httpmock.RegisterResponder(
				"GET",
				fakeScbeUrlApi+"/"+scbe.UrlScbeResourceService,
				httpmock.NewStringResponder(http.StatusOK, fakeServiceJsonResponse),
			)
			var services []scbe.ScbeStorageService
			_, err = client.Get(scbe.UrlScbeResourceService, nil, -1, &services)
			Expect(err).ToNot(HaveOccurred())
			Expect(services[0].Name).To(Equal("gold"))
		})
		It("should fail when Get succeed and return an expacted struct back", func() {
			httpmock.RegisterResponder(
				"GET",
				fakeScbeUrlApi+"/"+scbe.UrlScbeResourceService,
				httpmock.NewStringResponder(http.StatusBadRequest, fakeServiceJsonResponse),
			)
			var services []scbe.ScbeStorageService
			_, err = client.Get(scbe.UrlScbeResourceService, nil, -1, &services)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(MatchRegexp("^Error, bad status code"))
		})
		It("Login and retry rest call if token expired", func() {
			var numLogin, numGetServices int
			loginResponse := scbe.LoginResponse{Token: "fake-token-2"}
			marshalledResponse, err := json.Marshal(loginResponse)
			Expect(err).ToNot(HaveOccurred())
			httpmock.RegisterResponder("POST", fakeScbeUrlAuthFull,
				CountLoginResponder(&numLogin, string(marshalledResponse)))
			httpmock.RegisterResponder("GET", fakeScbeUrlApi+"/"+scbe.UrlScbeResourceService,
				TokenExpiredResponder(&numGetServices))
			var services []scbe.ScbeStorageService
			_, err = client.Get(scbe.UrlScbeResourceService, nil, http.StatusOK, &services)
			Expect(err).ToNot(HaveOccurred())
			Expect(numLogin).To(Equal(1))
			Expect(numGetServices).To(Equal(2))
		})
	})
})

var _ = Describe("ScbeRestClient", func() {
	var (
		logger         *log.Logger
		scbeRestClient scbe.ScbeRestClient
		fakeRestClient *fakes.FakeRestClient
		err            error
	)
	BeforeEach(func() {
		logger = log.New(os.Stdout, "ubiquity scbe: ", log.Lshortfile|log.LstdFlags)
		fakeRestClient = new(fakes.FakeRestClient)
		credentialInfo := resources.CredentialInfo{"user", "password", "flocker"}
		conInfo := resources.ConnectionInfo{credentialInfo, 8440, "ip", true}

		scbeRestClient, err = scbe.NewScbeRestClientWithNewRestClient(
			logger,
			conInfo,
			fakeRestClient,
		)
		Expect(err).ToNot(HaveOccurred())
		fakeRestClient.LoginReturns(nil) // Fake the login to SCBE

	})

	Context(".Get", func() {
		It("should fail to check if service exist when return err", func() {
			fakeRestClient.GetReturns(nil, fmt.Errorf("error"))
			exist, err := scbeRestClient.ServiceExist(fakeProfileName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("error"))
			Expect(exist).To(Equal(false))
		})
		It("should fail when service doesn't exist", func() {
			services := make([]scbe.ScbeStorageService, 1)
			services[0].Name = fakeProfileName
			services[0].Id = "666"

			fakeRestClient.GetReturns(services, nil)
			exist, err := scbeRestClient.ServiceExist("fake_profile_NOT_EXIST")
			Expect(err).NotTo(HaveOccurred())
			Expect(exist).To(Equal(false))
		})
		It("should succeed to check if service exist", func() {
			services := make([]scbe.ScbeStorageService, 1)
			services[0].Name = fakeProfileName
			services[0].Id = "666"

			fakeRestClient.GetReturns(services, nil)
			exist, err := scbeRestClient.ServiceExist(fakeProfileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exist).To(Equal(true))
		})
		It("should fail to create vol when service list fail", func() {
			fakeRestClient.GetReturns(nil, fmt.Errorf("error"))
			_, err := scbeRestClient.CreateVolume("fakevol", fakeProfileName, 10)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("error"))
		})
		It("should fail to create vol when service list return service name you did not expect", func() {
			services := make([]scbe.ScbeStorageService, 1)
			services[0].Name = fakeProfileName
			services[0].Id = "666"

			fakeRestClient.GetReturns(services, nil)
			_, err := scbeRestClient.CreateVolume("fakevol", "fake_profile_NOT_EXIST", 10)
			Expect(err).To(HaveOccurred())
		})
		It("should fail to create vol when post fail", func() {
			services := make([]scbe.ScbeStorageService, 1)
			services[0].Name = fakeProfileName
			services[0].Id = "666'"

			fakeRestClient.GetReturns(services, nil)
			fakeRestClient.PostReturns(fmt.Errorf("error666"))
			_, err := scbeRestClient.CreateVolume("fakevol", fakeProfileName, 10)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(MatchRegexp("error666"))
		})
		It("should succeed to create vol and return ScbeVolumeInfo object", func() {
			Skip("TBD")
			services := make([]scbe.ScbeStorageService, 1)
			services[0].Name = fakeProfileName
			services[0].Id = "666'"

			fakeRestClient.GetReturns(services, nil)
			fakeRestClient.PostReturns(nil)
			scbeVolumeInfo, err := scbeRestClient.CreateVolume("fakevol", fakeProfileName, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(scbeVolumeInfo.Name).To(Equal("fakevol"))
			Expect(scbeVolumeInfo.Wwn).To(Equal("www1"))
			Expect(scbeVolumeInfo.ServiceName).To(Equal(fakeProfileName))
		})

	})
})

func CountLoginResponder(num *int, loginResp string) httpmock.Responder {
	*num = 0
	return func(req *http.Request) (*http.Response, error) {
		*num++
		return httpmock.NewStringResponse(http.StatusOK, loginResp), nil
	}
}

func TokenExpiredResponder(num *int) httpmock.Responder {
	*num = 0
	return func(req *http.Request) (*http.Response, error) {
		*num++
		if *num == 1 {
			return httpmock.NewStringResponse(http.StatusUnauthorized, ""), nil
		} else {
			return httpmock.NewStringResponse(http.StatusOK, fakeServiceJsonResponse), nil
		}
	}
}
