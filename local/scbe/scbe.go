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

package scbe

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/midoblgsm/ubiquity/resources"
	"github.com/midoblgsm/ubiquity/utils"
	"github.com/midoblgsm/ubiquity/utils/logs"
	"strconv"
	"strings"
	"sync"
)

type scbeLocalClient struct {
	logger         logs.Logger
	dataModel      ScbeDataModel
	scbeRestClient ScbeRestClient
	isActivated    bool
	config         resources.ScbeConfig
	activationLock *sync.RWMutex
	locker         utils.Locker
}

const (
	OptionNameForServiceName = "profile"
	OptionNameForVolumeSize  = "size"
	volumeNamePrefix         = "u_"
	AttachedToNothing        = "" // during provisioning the volume is not attached to any host
	EmptyHost                = ""
	ComposeVolumeName        = volumeNamePrefix + "%s_%s" // e.g u_instance1_volName
	MaxVolumeNameLength      = 63                         // IBM block storage max volume name cannot exceed this length

	GetVolumeConfigExtraParams = 1 // number of extra params added to the VolumeConfig beyond the scbe volume struct
)

var (
	SupportedFSTypes = []string{"ext4", "xfs"}
)

func NewScbeLocalClient(config resources.ScbeConfig, database *gorm.DB) (resources.StorageClient, error) {
	logger := logs.GetLogger()
	datamodel := NewScbeDataModel(database, resources.SCBE)
	err := datamodel.CreateVolumeTable()
	if err != nil {
		return &scbeLocalClient{}, logger.ErrorRet(err, "failed")
	}
	scbeRestClient := NewScbeRestClient(config.ConnectionInfo)
	return NewScbeLocalClientWithNewScbeRestClientAndDataModel(config, datamodel, scbeRestClient)
}
func NewScbeLocalClientWithNewScbeRestClientAndDataModel(config resources.ScbeConfig, dataModel ScbeDataModel, scbeRestClient ScbeRestClient) (resources.StorageClient, error) {
	if err := validateScbeConfig(&config); err != nil {
		return &scbeLocalClient{}, err
	}

	client := &scbeLocalClient{
		logger:         logs.GetLogger(),
		scbeRestClient: scbeRestClient, // TODO need to mock it in more advance way
		dataModel:      dataModel,
		config:         config,
		activationLock: &sync.RWMutex{},
		locker:         utils.NewLocker(),
	}
	if err := basicScbeLocalClientStartupAndValidation(client); err != nil {
		return &scbeLocalClient{}, err
	}
	return client, nil
}

// basicScbeLocalClientStartup validate config params, login to SCBE and validate default exist
func basicScbeLocalClientStartupAndValidation(s *scbeLocalClient) error {
	if err := s.scbeRestClient.Login(); err != nil {
		return s.logger.ErrorRet(err, "scbeRestClient.Login() failed")
	}
	s.logger.Info("scbeRestClient.Login() succeeded", logs.Args{{"SCBE", s.config.ConnectionInfo.ManagementIP}})

	isExist, err := s.scbeRestClient.ServiceExist(s.config.DefaultService)
	if err != nil {
		return s.logger.ErrorRet(err, "scbeRestClient.ServiceExist failed")
	}

	if isExist == false {
		return s.logger.ErrorRet(&activateDefaultServiceError{s.config.DefaultService, s.config.ConnectionInfo.ManagementIP}, "failed")
	}
	s.logger.Info("The default service exist in SCBE", logs.Args{{s.config.ConnectionInfo.ManagementIP, s.config.DefaultService}})
	return nil
}

func validateScbeConfig(config *resources.ScbeConfig) error {
	logger := logs.GetLogger()

	if config.DefaultVolumeSize == "" {
		// means customer didn't configure the default
		config.DefaultVolumeSize = resources.DefaultForScbeConfigParamDefaultVolumeSize
		logger.Debug("No DefaultVolumeSize defined in conf file, so set the DefaultVolumeSize to value " + resources.DefaultForScbeConfigParamDefaultVolumeSize)
	}
	_, err := strconv.Atoi(config.DefaultVolumeSize)
	if err != nil {
		return logger.ErrorRet(&ConfigDefaultSizeNotNumError{}, "failed")
	}

	if config.DefaultFilesystemType == "" {
		// means customer didn't configure the default
		config.DefaultFilesystemType = resources.DefaultForScbeConfigParamDefaultFilesystem
		logger.Debug("No DefaultFileSystemType defined in conf file, so set the DefaultFileSystemType to value " + resources.DefaultForScbeConfigParamDefaultFilesystem)
	} else if !utils.StringInSlice(config.DefaultFilesystemType, SupportedFSTypes) {
		return logger.ErrorRet(
			&ConfigDefaultFilesystemTypeNotSupported{
				config.DefaultFilesystemType,
				strings.Join(SupportedFSTypes, ",")}, "failed")
	}

	if len(config.UbiquityInstanceName) > resources.UbiquityInstanceNameMaxSize {
		return logger.ErrorRet(&ConfigScbeUbiquityInstanceNameWrongSize{}, "failed")
	}
	// TODO add more verification on the config file.
	return nil
}

func (s *scbeLocalClient) Activate(activateRequest resources.ActivateRequest) resources.ActivateResponse {
	defer s.logger.Trace(logs.DEBUG)()
	s.activationLock.RLock()
	if s.isActivated {
		s.activationLock.RUnlock()
		return resources.ActivateResponse{}
	}
	s.activationLock.RUnlock()

	s.activationLock.Lock() //get a write lock to prevent others from repeating these actions
	defer s.activationLock.Unlock()

	// Nothing special to activate SCBE
	s.isActivated = true
	return resources.ActivateResponse{}
}

// CreateVolume parse and validate the given options and trigger the volume creation
func (s *scbeLocalClient) CreateVolume(createVolumeRequest resources.CreateVolumeRequest) resources.CreateVolumeResponse {
	defer s.logger.Trace(logs.DEBUG)()

	_, volExists, err := s.dataModel.GetVolume(createVolumeRequest.Name)
	if err != nil {
		return resources.CreateVolumeResponse{Error: s.logger.ErrorRet(err, "dataModel.GetVolume failed", logs.Args{{"name", createVolumeRequest.Name}})}
	}

	// validate volume doesn't exist
	if volExists {
		return resources.CreateVolumeResponse{Error: s.logger.ErrorRet(&volAlreadyExistsError{createVolumeRequest.Name}, "failed")}
	}

	// validate size option given
	sizeStr, ok := createVolumeRequest.Metadata[OptionNameForVolumeSize]
	if !ok {
		sizeStr = s.config.DefaultVolumeSize
		s.logger.Debug("No size given to create volume, so using the default_size",
			logs.Args{{"volume", createVolumeRequest.Name}, {"default_size", sizeStr}})
	}

	// validate size is a number
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return resources.CreateVolumeResponse{Error: s.logger.ErrorRet(&provisionParamIsNotNumberError{createVolumeRequest.Name, OptionNameForVolumeSize}, "failed")}
	}

	// validate fstype option given
	fstypeInt, ok := createVolumeRequest.Metadata[resources.OptionNameForVolumeFsType]
	var fstype string
	if !ok {
		fstype = s.config.DefaultFilesystemType
		s.logger.Debug("No default file system type given to create a volume, so using the default_fstype",
			logs.Args{{"volume", createVolumeRequest.Name}, {"default_fstype", fstype}})
	} else {
		fstype = fstypeInt
	}
	if !utils.StringInSlice(fstype, SupportedFSTypes) {
		return resources.CreateVolumeResponse{Error: s.logger.ErrorRet(
			&FsTypeNotSupportedError{createVolumeRequest.Name, fstype, strings.Join(SupportedFSTypes, ",")}, "failed")}
	}

	// Get the profile option
	profile := s.config.DefaultService
	if createVolumeRequest.Metadata[OptionNameForServiceName] != "" && createVolumeRequest.Metadata[OptionNameForServiceName] != "" {
		profile = createVolumeRequest.Metadata[OptionNameForServiceName]
	}

	// Generate the designated volume name by template
	volNameToCreate := fmt.Sprintf(ComposeVolumeName, s.config.UbiquityInstanceName, createVolumeRequest.Name)

	// Validate volume length ok
	volNamePrefixForCheckLength := fmt.Sprintf(ComposeVolumeName, s.config.UbiquityInstanceName, "")
	volNamePrefixForCheckLengthLen := len(volNamePrefixForCheckLength)
	if len(volNameToCreate) > MaxVolumeNameLength {
		maxVolLength := MaxVolumeNameLength - volNamePrefixForCheckLengthLen // its dynamic because it depends on the UbiquityInstanceName len
		return resources.CreateVolumeResponse{Error: s.logger.ErrorRet(&VolumeNameExceededMaxLengthError{createVolumeRequest.Name, maxVolLength}, "failed")}
	}
	//TODO: check this
	metadata := resources.VolumeMetadata{Values: createVolumeRequest.Metadata}
	volume := resources.Volume{Name: createVolumeRequest.Name, CapacityBytes: createVolumeRequest.CapacityBytes, Metadata: metadata, Backend: createVolumeRequest.Backend}
	// Provision the volume on SCBE service
	volInfo := ScbeVolumeInfo{}
	volInfo, err = s.scbeRestClient.CreateVolume(volNameToCreate, profile, size)
	if err != nil {
		return resources.CreateVolumeResponse{Error: s.logger.ErrorRet(err, "scbeRestClient.CreateVolume failed")}
	}

	err = s.dataModel.InsertVolume(createVolumeRequest.Name, volInfo.Wwn, AttachedToNothing, fstype)
	if err != nil {
		return resources.CreateVolumeResponse{Error: s.logger.ErrorRet(err, "dataModel.InsertVolume failed")}
	}

	s.logger.Info("succeeded", logs.Args{{"volume", createVolumeRequest.Name}, {"profile", profile}})
	return resources.CreateVolumeResponse{Volume: volume}
}

func (s *scbeLocalClient) RemoveVolume(removeVolumeRequest resources.RemoveVolumeRequest) resources.RemoveVolumeResponse {
	defer s.logger.Trace(logs.DEBUG)()

	existingVolume, volExists, err := s.dataModel.GetVolume(removeVolumeRequest.Name)
	if err != nil {
		return resources.RemoveVolumeResponse{Error: s.logger.ErrorRet(err, "dataModel.GetVolume failed")}
	}

	if volExists == false {
		return resources.RemoveVolumeResponse{Error: s.logger.ErrorRet(fmt.Errorf("Volume [%s] not found", removeVolumeRequest.Name), "failed")}
	}
	if existingVolume.AttachTo != EmptyHost {
		return resources.RemoveVolumeResponse{Error: s.logger.ErrorRet(&CannotDeleteVolWhichAttachedToHostError{removeVolumeRequest.Name, existingVolume.AttachTo}, "failed")}
	}

	if err = s.scbeRestClient.DeleteVolume(existingVolume.WWN); err != nil {
		return resources.RemoveVolumeResponse{Error: s.logger.ErrorRet(err, "scbeRestClient.DeleteVolume failed")}
	}

	if err = s.dataModel.DeleteVolume(removeVolumeRequest.Name); err != nil {
		return resources.RemoveVolumeResponse{Error: s.logger.ErrorRet(err, "dataModel.DeleteVolume failed")}
	}

	return resources.RemoveVolumeResponse{}
}

func (s *scbeLocalClient) GetVolume(getVolumeRequest resources.GetVolumeRequest) resources.GetVolumeResponse {
	defer s.logger.Trace(logs.DEBUG)()

	existingVolume, volExists, err := s.dataModel.GetVolume(getVolumeRequest.Name)
	if err != nil {
		return resources.GetVolumeResponse{Error: s.logger.ErrorRet(err, "dataModel.GetVolume failed")}
	}
	if volExists == false {
		return resources.GetVolumeResponse{Error: s.logger.ErrorRet(errors.New("Volume not found"), "failed")}
	}

	return resources.GetVolumeResponse{Volume: existingVolume.Volume}
}

func (s *scbeLocalClient) GetVolumeConfig(getVolumeConfigRequest resources.GetVolumeConfigRequest) resources.GetVolumeConfigResponse {
	defer s.logger.Trace(logs.DEBUG)()

	// get volume wwn from name
	scbeVolume, volExists, err := s.dataModel.GetVolume(getVolumeConfigRequest.Name)
	if err != nil {
		return resources.GetVolumeConfigResponse{Error: s.logger.ErrorRet(err, "dataModel.GetVolume failed")}
	}

	// verify volume exists
	if !volExists {
		return resources.GetVolumeConfigResponse{Error: s.logger.ErrorRet(errors.New("Volume not found"), "failed")}
	}

	// get volume full info from scbe
	volumeInfo, err := s.scbeRestClient.GetVolumes(scbeVolume.WWN)
	if err != nil {
		return resources.GetVolumeConfigResponse{Error: s.logger.ErrorRet(err, "scbeRestClient.GetVolumes failed")}
	}

	// verify volume is found
	if len(volumeInfo) != 1 {
		return resources.GetVolumeConfigResponse{Error: s.logger.ErrorRet(&volumeNotFoundError{getVolumeConfigRequest.Name}, "failed", logs.Args{{"volumeInfo", volumeInfo}})}
	}

	// serialize scbeVolumeInfo to json
	jsonData, err := json.Marshal(volumeInfo[0])
	if err != nil {
		return resources.GetVolumeConfigResponse{Error: s.logger.ErrorRet(err, "json.Marshal failed")}
	}

	// convert json to map[string]interface{}
	var volConfig map[string]interface{}
	if err = json.Unmarshal(jsonData, &volConfig); err != nil {
		return resources.GetVolumeConfigResponse{Error: s.logger.ErrorRet(err, "json.Unmarshal failed")}
	}

	// The ubiquity remote will use this extra info to determine the fstype needed to be created on this volume while attaching
	volConfig[resources.OptionNameForVolumeFsType] = scbeVolume.FSType
	return resources.GetVolumeConfigResponse{VolumeConfig: volConfig}
}

func (s *scbeLocalClient) Attach(attachRequest resources.AttachRequest) resources.AttachResponse {
	defer s.logger.Trace(logs.DEBUG)()

	if attachRequest.Host == EmptyHost {
		return resources.AttachResponse{Error: s.logger.ErrorRet(
			&InValidRequestError{"attachRequest", "Host", attachRequest.Host, "none empty string"}, "failed")}
	}
	if attachRequest.Name == "" {
		return resources.AttachResponse{Error: s.logger.ErrorRet(
			&InValidRequestError{"attachRequest", "Name", attachRequest.Name, "none empty string"}, "failed")}
	}

	existingVolume, volExists, err := s.dataModel.GetVolume(attachRequest.Name)
	if err != nil {
		return resources.AttachResponse{Error: s.logger.ErrorRet(err, "dataModel.GetVolume failed")}
	}

	if !volExists {
		return resources.AttachResponse{Error: s.logger.ErrorRet(&volumeNotFoundError{attachRequest.Name}, "failed")}
	}

	if existingVolume.AttachTo == attachRequest.Host {
		// if already map to the given host then just ignore and succeed to attach
		s.logger.Info("Volume already attached, skip backend attach", logs.Args{{"volume", attachRequest.Name}, {"host", attachRequest.Host}})
		volumeMountpoint := fmt.Sprintf(resources.PathToMountUbiquityBlockDevices, existingVolume.WWN)
		return resources.AttachResponse{Mountpoint: volumeMountpoint}
	} else if existingVolume.AttachTo != "" {
		return resources.AttachResponse{Error: s.logger.ErrorRet(&volAlreadyAttachedError{attachRequest.Name, existingVolume.AttachTo}, "failed")}
	}

	// Lock will ensure no other caller attach a volume from the same host concurrently, Prevent SCBE race condition on get next available lun ID
	s.locker.WriteLock(attachRequest.Host)
	s.logger.Debug("Attaching", logs.Args{{"volume", existingVolume}})
	if _, err = s.scbeRestClient.MapVolume(existingVolume.WWN, attachRequest.Host); err != nil {
		s.locker.WriteUnlock(attachRequest.Host)
		return resources.AttachResponse{Error: s.logger.ErrorRet(err, "scbeRestClient.MapVolume failed")}
	}
	s.locker.WriteUnlock(attachRequest.Host)

	if err = s.dataModel.UpdateVolumeAttachTo(attachRequest.Name, existingVolume, attachRequest.Host); err != nil {
		return resources.AttachResponse{Error: s.logger.ErrorRet(err, "dataModel.UpdateVolumeAttachTo failed")}
	}

	volumeMountpoint := fmt.Sprintf(resources.PathToMountUbiquityBlockDevices, existingVolume.WWN)
	return resources.AttachResponse{Mountpoint: volumeMountpoint}
}

func (s *scbeLocalClient) Detach(detachRequest resources.DetachRequest) resources.DetachResponse {
	defer s.logger.Trace(logs.DEBUG)()
	host2detach := detachRequest.Host

	existingVolume, volExists, err := s.dataModel.GetVolume(detachRequest.Name)
	if err != nil {
		return resources.DetachResponse{Error: s.logger.ErrorRet(err, "dataModel.GetVolume failed")}
	}

	if !volExists {
		return resources.DetachResponse{Error: s.logger.ErrorRet(&volumeNotFoundError{detachRequest.Name}, "failed")}
	}

	// Fail if vol already detach
	if existingVolume.AttachTo == EmptyHost {
		return resources.DetachResponse{Error: s.logger.ErrorRet(&volNotAttachedError{detachRequest.Name}, "failed")}
	}

	s.logger.Debug("Detaching", logs.Args{{"volume", existingVolume}})
	if err = s.scbeRestClient.UnmapVolume(existingVolume.WWN, host2detach); err != nil {
		return resources.DetachResponse{Error: s.logger.ErrorRet(err, "scbeRestClient.UnmapVolume failed")}
	}

	if err = s.dataModel.UpdateVolumeAttachTo(detachRequest.Name, existingVolume, EmptyHost); err != nil {
		return resources.DetachResponse{Error: s.logger.ErrorRet(err, "dataModel.UpdateVolumeAttachTo failed")}
	}

	return resources.DetachResponse{}
}

func (s *scbeLocalClient) ListVolumes(listVolumesRequest resources.ListVolumesRequest) resources.ListVolumesResponse {
	defer s.logger.Trace(logs.DEBUG)()
	var err error

	volumesInDb, err := s.dataModel.ListVolumes()
	if err != nil {
		return resources.ListVolumesResponse{Error: s.logger.ErrorRet(err, "dataModel.ListVolumes failed")}
	}

	s.logger.Debug("Volumes in db", logs.Args{{"num", len(volumesInDb)}})
	var volumes []resources.Volume
	for _, volume := range volumesInDb {
		s.logger.Debug("Volumes from db", logs.Args{{"volume", volume}})
		volumes = append(volumes, volume.Volume)
	}

	return resources.ListVolumesResponse{Volumes: volumes}
}

func (s *scbeLocalClient) getVolumeMountPoint(volume ScbeVolume) (string, error) {
	defer s.logger.Trace(logs.DEBUG)()

	//TODO return mountpoint
	return "some mount point", nil
}
