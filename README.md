# Ubiquity Storage Service for Container Ecosystems 
[![Build Status](https://travis-ci.org/midoblgsm/ubiquity.svg?branch=master)](https://travis-ci.org/midoblgsm/ubiquity)
[![GoDoc](https://godoc.org/github.com/midoblgsm/ubiquity?status.svg)](https://godoc.org/github.com/midoblgsm/ubiquity)
[![Coverage Status](https://coveralls.io/repos/github/midoblgsm/ubiquity/badge.svg?branch=master)](https://coveralls.io/github/midoblgsm/ubiquity?branch=master)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/midoblgsm/ubiquity)](https://goreportcard.com/report/github.com/midoblgsm/ubiquity)

The Ubiquity project enables persistent storage for CSI compliant container frameworks. 
It is a pluggable framework available for different storage systems. The framework interfaces with the storage systems, using their plugins. The [Available Storage Systems](supportedStorage.md) section describes the storage system  configuration and deployment options. Different container frameworks can use Ubiquity concurrently, allowing access to different storage systems. 


![Ubiquity Overview](images/UbiquityOverview.jpg)

Ubiquity supports the Kubernetes and Docker frameworks, using the following plugins:

- [Ubiquity Docker volume plugin](https://github.com/IBM/ubiquity-docker-plugin)
- [Ubiquity plugin for Kubernetes](https://github.com/IBM/ubiquity-k8s) (Dynamic Provisioner and FlexVolume)

The code is provided as is, without warranty. Any issue will be handled on a best-effort basis.

## Installing the Ubiquity service

### Prerequisites
  * Install [golang](https://golang.org/) (>=1.6).
  * Install [git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git).
  * Configure go. GOPATH environment variable must be set correctly before starting the build process. Create a new directory and set it as GOPATH.

### Download and build source code
* Build Ubiquity service from source. 
```bash
mkdir -p $HOME/workspace
export GOPATH=$HOME/workspace
mkdir -p $GOPATH/src/github.com/midoblgsm
cd $GOPATH/src/github.com/midoblgsm
git clone git@github.com:midoblgsm/ubiquity.git
cd ubiquity-csi
./scripts/run_glide_up
./scripts/build
```
The built binary will be in the bin directory inside the repository folder.
To run it you need to setup ubiquity configuration in [ubiquity-server.conf](ubiquity-server.conf).
After that start the server:
```bash
./bin/ubiquity
```

### Running unit tests for ubiquity-csi

Install these go packages to test Ubiquity:
```bash
# Install ginkgo
go install github.com/onsi/ginkgo/ginkgo
# Install gomega
go install github.com/onsi/gomega
```

Run the tests:
```bash
./scripts/run-unit-tests
```


### Configuring the Ubiquity service for storage other then localhost
Before running the Ubiquity service, you must create and configure the `/etc/ubiquity/ubiquity-server.conf` file, according to your storage system type.
Follow the configuration procedures detailed in the [Available Storage Systems](supportedStorage.md) section.


###  Running the Ubiquity service
  * Run the service.
```bash
./bin/ubiquity
```


### Installing Ubiquity plugins for CSI
To use the active Ubiquity service, install Ubiquity CSI plugin.
  * [Ubiquity CSI plugin](https://github.com/midoblgsm/ubiquity-csi)


### Contribution
To contribute, follow the guidelines in [Contribution guide](contribution-guide.md)


### Troubleshooting
* Review the Ubiquity logs for any issues:
    * [logPath]/ubiquity.log   ([logPath] configured in the ubiquity-server.conf)
    * /var/log/messages        

### Support
For any questions, suggestions, or issues, use github.


