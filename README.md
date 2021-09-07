# vagrant_tools

# Description
Creates and manages vm's according to a configfile in yaml using vagrant and it's plugins.

Outputs an inventory yaml file for ansible.

Tested on libvirt (native Linux VM, KVM etc.), VMware ESXi, and Microsoft HyperV with varying levels of frequency.


# Usage
## libvirt / KVM
Assumes you run locally on a Linux machine that hosts the vm's
EXAMPLE:
```
./vagrant_tools --configfile configs/sample.yaml --action up
```
## VMware ESXi
Requires you run from a machine that can talk to a targeted ESXi servers management network.
```
./vagrant_tools --configfile configs/esxi_sample.yaml --esxi_hostname 10.10.10.10 --esxi_username spongebob --esxi_password crabbypatty
```
NOTE: It is recommended you use the Docker container, see below.
ADDITIONALLY: Per VMware docs, When manually specifying mac addresses for esxi they must be in the range
              00:50:56:00:00:00 - 00:50:56:3f:ff:ff or the host will reject the NIC!


## HyperV
Obvious, requires running on Windows.
```
vagrant_tools.exe --configfile configs/sample.yaml --action up
```


# Building
## Download
```
cd $GOPATH/src/
git clone git@github.com:resurgentech/vagrant_tools.git
```
## Build
```
cd $GOPATH/src/vagrant_tools
go get
go build
```
## Install
After build.
```
go install
```


# Docker Container
Included is Dockerfile for building an image that can build this code and running the esxi provider.

NOTE: Download the VMware-ovftool-XXXXXX to ./downloads in this working copy.  Update the version in the Dockerfile as needed

## Build
```
docker build . --tag vagrant_tools
```
## Running
```
docker run -it --rm --network=host -v `pwd`:/mnt -w /mnt vagrant_tools:latest vagrant_tools -configfile configs/esxi_sample.yaml -action up -esxi_hostname 10.10.10.10 -esxi_username spongebob -esxi_password crabbypatty
```


# Manual installing client for esxi vagrant
Setting up for using vagrant on esxi requires a little special handling for getting the client setup for interacting with the esxi system.  These worked last time we did it, but versions and such change from time to time.
```
wget https://releases.hashicorp.com/vagrant/2.2.5/vagrant_2.2.5_x86_64.deb
sudo dpkg -i vagrant_2.2.5_x86_64.deb
rm -rf vagrant_2.2.5_x86_64.deb
sudo apt-get install -y build-essential
vagrant plugin install vagrant-vmware-esxi
vagrant plugin install vagrant-reload
```
## Installing ovftool
Download ovftool from VMware, either use this version or the latest working.
`sudo ./downloads/VMware-ovftool-4.3.0-13981069-lin.x86_64.bundle`
