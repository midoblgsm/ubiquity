/**
 * Copyright 2017 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package localhost

import (
	"log"

	"os"
	"path"

	"fmt"

	"github.com/jinzhu/gorm"

	"sync"

	"github.com/midoblgsm/ubiquity/resources"
)

type localhostLocalClient struct {
	logger         *log.Logger
	dataModel      LocalhostDataModel
	config         resources.LocalHostConfig
	isActivated    bool
	activationLock *sync.RWMutex
}

func NewLocalhostLocalClient(logger *log.Logger, config resources.UbiquityServerConfig, database *gorm.DB) (resources.StorageClient, error) {

	return newLocalhostLocalClient(logger, config.LocalHostConfig, database, resources.LocalHost)
}

func newLocalhostLocalClient(logger *log.Logger, config resources.LocalHostConfig, database *gorm.DB, backend string) (*localhostLocalClient, error) {
	logger.Println("localhostLocalClient: init start")
	defer logger.Println("localhostLocalClient: init end")

	datamodel := NewLocalhostDataModel(logger, database, backend)
	err := datamodel.CreateVolumeTable()
	if err != nil {
		return &localhostLocalClient{}, err
	}
	return &localhostLocalClient{logger: logger, config: config, dataModel: datamodel, activationLock: &sync.RWMutex{}}, nil
}

func (s *localhostLocalClient) Activate(activateRequest resources.ActivateRequest) resources.ActivateResponse {
	s.logger.Println("localhostLocalClient: Activate start")
	defer s.logger.Println("localhostLocalClient: Activate end")

	s.activationLock.RLock()
	if s.isActivated {
		s.activationLock.RUnlock()
		return resources.ActivateResponse{}
	}
	s.activationLock.RUnlock()

	s.activationLock.Lock() //get a write lock to prevent others from repeating these actions
	defer s.activationLock.Unlock()

	err := os.MkdirAll(s.config.LocalhostPath, 0777)
	if err != nil {
		s.logger.Println(err.Error())
		return resources.ActivateResponse{Error: err}
	}
	s.isActivated = true
	return resources.ActivateResponse{}
}

func (s *localhostLocalClient) CreateVolume(createVolumeRequest resources.CreateVolumeRequest) resources.CreateVolumeResponse {
	s.logger.Println("localhostLocalClient: create start")
	defer s.logger.Println("localhostLocalClient: create end")

	existingVolume, volExists, err := s.dataModel.GetVolume(createVolumeRequest.Name)

	if err != nil {
		s.logger.Println(err.Error())
		return resources.CreateVolumeResponse{Error: err}
	}

	if volExists {
		return resources.CreateVolumeResponse{Volume: existingVolume}
	}

	s.logger.Printf("Opts for create: %#v\n", createVolumeRequest.Metadata)
	metadata := resources.VolumeMetadata{Values: createVolumeRequest.Metadata}
	volume := resources.Volume{Name: createVolumeRequest.Name, Backend: createVolumeRequest.Backend, Metadata: metadata, CapacityBytes: createVolumeRequest.CapacityBytes}
	volumePath := path.Join(s.config.LocalhostPath, volume.Name)
	err = os.MkdirAll(volumePath, 0777)
	if err != nil {
		s.logger.Println(err.Error())
		return resources.CreateVolumeResponse{Error: err}
	}

	s.dataModel.InsertVolume(volume)
	return resources.CreateVolumeResponse{Volume: volume}
}

func (s *localhostLocalClient) RemoveVolume(removeVolumeRequest resources.RemoveVolumeRequest) resources.RemoveVolumeResponse {
	s.logger.Println("localhostLocalClient: remove start")
	defer s.logger.Println("localhostLocalClient: remove end")

	existingVolume, volExists, err := s.dataModel.GetVolume(removeVolumeRequest.Name)

	if err != nil {
		s.logger.Println(err.Error())
		return resources.RemoveVolumeResponse{Error: err}
	}

	if volExists == false {
		return resources.RemoveVolumeResponse{Error: fmt.Errorf("Volume not found")}
	}

	pathToDel := path.Join(s.config.LocalhostPath, existingVolume.Name)
	err = os.RemoveAll(pathToDel)
	if err != nil {
		s.logger.Println(err.Error())
		return resources.RemoveVolumeResponse{Error: err}
	}

	err = os.RemoveAll(existingVolume.Mountpoint)
	if err != nil {
		s.logger.Println(err.Error())
		return resources.RemoveVolumeResponse{Error: err}
	}

	err = s.dataModel.DeleteVolume(removeVolumeRequest.Name)

	if err != nil {
		s.logger.Println(err.Error())
		return resources.RemoveVolumeResponse{Error: err}
	}

	return resources.RemoveVolumeResponse{}
}

func (s *localhostLocalClient) GetVolume(getVolumeRequest resources.GetVolumeRequest) resources.GetVolumeResponse {
	s.logger.Println("localhostLocalClient: GetVolume start")
	defer s.logger.Println("localhostLocalClient: GetVolume finish")

	existingVolume, volExists, err := s.dataModel.GetVolume(getVolumeRequest.Name)
	if err != nil {
		return resources.GetVolumeResponse{Error: err}
	}
	if volExists == false {
		return resources.GetVolumeResponse{Error: fmt.Errorf("Volume not found")}
	}

	return resources.GetVolumeResponse{Volume: existingVolume}
}

func (s *localhostLocalClient) GetVolumeConfig(getVolumeConfigRequest resources.GetVolumeConfigRequest) resources.GetVolumeConfigResponse {
	s.logger.Println("localhostLocalClient: GetVolumeConfig start")
	defer s.logger.Println("localhostLocalClient: GetVolumeConfig finish")

	existingVolume, volExists, err := s.dataModel.GetVolume(getVolumeConfigRequest.Name)

	if err != nil {
		s.logger.Println(err.Error())
		return resources.GetVolumeConfigResponse{Error: err}
	}

	if volExists {
		volumeConfigDetails := make(map[string]interface{})
		volumeConfigDetails["mountpoint"] = existingVolume.Mountpoint
		if err != nil {
			s.logger.Println(err.Error())
			return resources.GetVolumeConfigResponse{Error: err}
		}

		return resources.GetVolumeConfigResponse{VolumeConfig: volumeConfigDetails}
	}
	return resources.GetVolumeConfigResponse{Error: fmt.Errorf("Volume not found")}
}

func (s *localhostLocalClient) Attach(attachRequest resources.AttachRequest) resources.AttachResponse {
	s.logger.Println("localhostLocalClient: attach start")
	defer s.logger.Println("localhostLocalClient: attach end")

	_, volExists, err := s.dataModel.GetVolume(attachRequest.Name)

	if err != nil {
		s.logger.Println(err.Error())
		return resources.AttachResponse{Error: err}
	}

	if !volExists {
		return resources.AttachResponse{Error: fmt.Errorf("Volume not found")}
	}

	return resources.AttachResponse{}
}

func (s *localhostLocalClient) Detach(detachRequest resources.DetachRequest) resources.DetachResponse {
	s.logger.Println("localhostLocalClient: detach start")
	defer s.logger.Println("localhostLocalClient: detach end")

	_, volExists, err := s.dataModel.GetVolume(detachRequest.Name)

	if err != nil {
		s.logger.Println(err.Error())
		return resources.DetachResponse{Error: err}
	}

	if !volExists {
		return resources.DetachResponse{Error: fmt.Errorf("Volume not found")}
	}

	return resources.DetachResponse{}
}

func (s *localhostLocalClient) ListVolumes(listVolumesRequest resources.ListVolumesRequest) resources.ListVolumesResponse {
	s.logger.Println("localhostLocalClient: list start")
	defer s.logger.Println("localhostLocalClient: list end")
	var err error

	volumesInDb, err := s.dataModel.ListVolumes()

	if err != nil {
		s.logger.Printf("error retrieving volumes from db %#v\n", err)
		return resources.ListVolumesResponse{Error: err}
	}

	return resources.ListVolumesResponse{Volumes: volumesInDb}
}
