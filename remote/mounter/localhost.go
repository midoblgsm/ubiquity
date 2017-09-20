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

package mounter

import (
	"fmt"
	"log"

	"github.com/midoblgsm/ubiquity/resources"
	"os"
	"path"
)

type localhostMounter struct {
	logger *log.Logger
	config resources.LocalHostConfig
}

func NewLocalHostMounter(logger *log.Logger, config resources.LocalHostConfig) resources.Mounter {
	return &localhostMounter{logger: logger, config: config}
}

func (s *localhostMounter) Mount(mountRequest resources.MountRequest) resources.MountResponse {
	s.logger.Println("localhostMounter: Mount start")
	defer s.logger.Println("localhostMounter: Mount end")

	volumeName, ok := mountRequest.VolumeConfig["volumeName"]

	if !ok {
		s.logger.Printf("volumeName should be specified %#v \n", mountRequest)
		return resources.MountResponse{Error: fmt.Errorf("volumeName not specified")}
	}

	volumePath := path.Join(s.config.LocalhostPath, fmt.Sprintf("%v", volumeName))
	err := os.Link(volumePath, mountRequest.Mountpoint)

	if err != nil {
		s.logger.Printf("Error removing mountpoint %#v \n", err)
		return resources.MountResponse{Error: err}
	}

	return resources.MountResponse{Mountpoint: mountRequest.Mountpoint}
}

func (s *localhostMounter) Unmount(unmountRequest resources.UnmountRequest) resources.UnmountResponse {
	s.logger.Println("spectrumScaleMounter: Unmount start")
	defer s.logger.Println("spectrumScaleMounter: Unmount end")

	// for spectrum-scale native: No Op for now
	return resources.UnmountResponse{}

}

func (s *localhostMounter) ActionAfterDetach(request resources.AfterDetachRequest) resources.AfterDetachResponse {
	// no action needed for SSc
	return resources.AfterDetachResponse{}
}
