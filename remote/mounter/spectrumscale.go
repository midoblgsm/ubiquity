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
	"github.com/midoblgsm/ubiquity/utils"
)

type spectrumScaleMounter struct {
	logger   *log.Logger
	executor utils.Executor
}

func NewSpectrumScaleMounter(logger *log.Logger) resources.Mounter {
	return &spectrumScaleMounter{logger: logger, executor: utils.NewExecutor()}
}

func (s *spectrumScaleMounter) Mount(mountRequest resources.MountRequest) resources.MountResponse {
	s.logger.Println("spectrumScaleMounter: Mount start")
	defer s.logger.Println("spectrumScaleMounter: Mount end")

	isPreexisting, isPreexistingSpecified := mountRequest.VolumeConfig["isPreexisting"]
	if isPreexistingSpecified && isPreexisting.(bool) == false {
		uid, uidSpecified := mountRequest.VolumeConfig["uid"]
		gid, gidSpecified := mountRequest.VolumeConfig["gid"]

		if uidSpecified || gidSpecified {
			args := []string{"chown", fmt.Sprintf("%s:%s", uid, gid), mountRequest.Mountpoint}
			_, err := s.executor.Execute("sudo", args)
			if err != nil {
				s.logger.Printf("Failed to change permissions of mountpoint %s: %s", mountRequest.Mountpoint, err.Error())
				return resources.MountResponse{Error: err}
			}
			//set permissions to specific user
			args = []string{"chmod", "og-rw", mountRequest.Mountpoint}
			_, err = s.executor.Execute("sudo", args)
			if err != nil {
				s.logger.Printf("Failed to set user permissions of mountpoint %s: %s", mountRequest.Mountpoint, err.Error())
				return resources.MountResponse{Error: err}
			}
		} else {
			//chmod 777 mountpoint
			args := []string{"chmod", "777", mountRequest.Mountpoint}
			_, err := s.executor.Execute("sudo", args)
			if err != nil {
				s.logger.Printf("Failed to change permissions of mountpoint %s: %s", mountRequest.Mountpoint, err.Error())
				return resources.MountResponse{Error: err}
			}
		}
	}

	return resources.MountResponse{Mountpoint: mountRequest.Mountpoint}
}

func (s *spectrumScaleMounter) Unmount(unmountRequest resources.UnmountRequest) resources.UnmountResponse {
	s.logger.Println("spectrumScaleMounter: Unmount start")
	defer s.logger.Println("spectrumScaleMounter: Unmount end")

	// for spectrum-scale native: No Op for now
	return resources.UnmountResponse{}

}

func (s *spectrumScaleMounter) ActionAfterDetach(request resources.AfterDetachRequest) resources.AfterDetachResponse {
	// no action needed for SSc
	return resources.AfterDetachResponse{}
}
