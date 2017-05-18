package scbe

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/IBM/ubiquity/resources"
	"github.com/IBM/ubiquity/utils"
)

/// SCBE rest client

//go:generate counterfeiter -o ../fakes/fake_scbe_rest_client.go . ScbeRestClient
type ScbeRestClient interface {
	Login() error
	CreateVolume(volName string, serviceName string, size_byte int) (ScbeVolumeInfo, error)
	GetAllVolumes() ([]ScbeVolumeInfo, error)
	GetVolume(wwn string) (ScbeVolumeInfo, error)
	DeleteVolume(wwn string) error
	MapVolume(wwn string, host string) error
	UnmapVolume(wwn string, host string) error
	GetVolMapping(wwn string) (string, error)
	ServiceExist(serviceName string) (bool, error)
}

type scbeRestClient struct {
	logger         *log.Logger
	baseURL        string
	authURL        string
	referrer       string
	connectionInfo resources.ConnectionInfo
	httpClient     *http.Client
	token          string
	headers        map[string]string
}

func NewScbeRestClient(logger *log.Logger, conInfo resources.ConnectionInfo) (ScbeRestClient, error) {
	// Set default SCBE port if not mentioned
	if conInfo.Port == 0 {
		conInfo.Port = DEFAULT_SCBE_PORT
	}
	// Add the default SCBE Flocker group to the credentials # TODO need to update with ubiquity group later on

	conInfo.CredentialInfo.Group = SCBE_FLOCKER_GROUP_PARAM
	referrer := fmt.Sprintf(URL_SCBE_REFERER, conInfo.ManagementIP, conInfo.Port)
	baseURL := referrer + URL_SCBE_BASE_SUFFIX
	headers := map[string]string{
		"Content-Type": "application/json",
		"referer":      referrer,
	}
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates TODO to use
	}
	client := &http.Client{Transport: transCfg}

	return &scbeRestClient{logger: logger,
		baseURL:        baseURL,
		authURL:        URL_SCBE_RESOURCE_GET_AUTH,
		referrer:       referrer,
		connectionInfo: conInfo,
		httpClient:     client,
		headers:        headers}, nil
}

func (s *scbeRestClient) Login() error {
	token, err := s.getToken()
	if err != nil {
		s.logger.Printf("Error in getting token %#v", err)
		return err
	}
	if token == "" {
		s.logger.Printf("Error, token is empty")
		return fmt.Errorf("Error, token is empty")
	}
	s.headers[HTTP_AUTH_KEY] = "Token " + token

	return nil

}

func (s *scbeRestClient) CreateVolume(volName string, serviceName string, size_byte int) (ScbeVolumeInfo, error) {
	return ScbeVolumeInfo{}, nil
}
func (s *scbeRestClient) GetAllVolumes() ([]ScbeVolumeInfo, error) {
	return nil, nil
}
func (s *scbeRestClient) GetVolume(wwn string) (ScbeVolumeInfo, error) {
	return ScbeVolumeInfo{}, nil
}
func (s *scbeRestClient) DeleteVolume(wwn string) error {
	return nil
}

func (s *scbeRestClient) MapVolume(wwn string, host string) error {
	return nil

}
func (s *scbeRestClient) UnmapVolume(wwn string, host string) error {
	return nil

}
func (s *scbeRestClient) GetVolMapping(wwn string) (string, error) {
	return "", nil
}

func (s *scbeRestClient) ServiceExist(serviceName string) (exist bool, err error) {
	var services []ScbeStorageService
	services, err = s.serviceList(serviceName)
	if err == nil {
		exist = len(services) > 0
	}
	return
}

func (s *scbeRestClient) serviceList(serviceName string) ([]ScbeStorageService, error) {
	payload := make(map[string]string)
	var err error
	if serviceName == "" {
		payload = nil // TODO else
	} else {
		payload["name"] = serviceName
	}
	url := utils.FormatURL(s.baseURL, UrlScbeResourceService)
	response, err := utils.HttpExecute(s.httpClient, s.logger, "GET", url, payload)
	if err != nil {
		return nil, err
	}
	err = utils.VerifyStatusCode(response.StatusCode, -1)
	if err != nil {
		return nil, err
	}
	var services []ScbeStorageService
	err = utils.UnmarshalResponse(response, services)
	if err != nil {
		return nil, err
	}

	return services, nil
}

func (s *scbeRestClient) getToken() (string, error) {
	delete(s.headers, HTTP_AUTH_KEY) // because no need token to get the token only user\password

	url := fmt.Sprintf("%s%s", s.baseURL, s.authURL)

	fmt.Printf("url*%s* \n", url)
	// response, err := utils.HttpExecute(s.httpClient, s.logger, "POST", url, s.connectionInfo.CredentialInfo)
	response, err := utils.HttpExecute(s.httpClient, s.logger, "POST", url, nil) //, s.connectionInfo.CredentialInfo)
	if err != nil {
		s.logger.Printf("failed executing http request %#v", err)
		return "", fmt.Errorf("failed executing http request %#v", err)
	}
	fmt.Printf("response ... %#v ", response)
	err = utils.VerifyStatusCode(response.StatusCode, http.StatusOK)
	if err != nil {
		return "", err
	}
	var services []ScbeStorageService
	err = utils.UnmarshalResponse(response, services)
	if err != nil {
		return "", err
	}

	var loginResponse = LoginResponse{}
	err = utils.UnmarshalResponse(response, &loginResponse)
	if err != nil {
		return "", err
	}

	if loginResponse.Token == "" {
		return "", fmt.Errorf("Token is empty")
	}

	return loginResponse.Token, nil
}
