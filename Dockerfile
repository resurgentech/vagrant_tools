FROM ubuntu

# install Ansible
COPY ./ansible /ansible
COPY ./downloads /ansible/downloads
RUN apt update; \
    apt install -y ansible; \
    cd /ansible ; \
    ansible-galaxy install -r requirements.yml; \
    ansible-playbook --connection=local -i inventory.localhost.yml vagrant.yml; \
    /ansible/downloads/VMware-ovftool-4.3.0-13981069-lin.x86_64.bundle --eulas-agreed --console --required; \
    rm -rf /ansible; \
    rm -rf /root/.ansible ; \
    apt-get clean

COPY vagrant_tools.go /
RUN . /etc/profile.d/golang.sh; mkdir -p $GOPATH/src/vagrant_tools/; \
    mv /vagrant_tools.go $GOPATH/src/vagrant_tools/; \
    cd $GOPATH/src/vagrant_tools/; \
    go get; \
    go build; \
    mv vagrant_tools /bin

RUN rm -rf /root/.vagrant.d/boxes/; mkdir -p /mnt/.vagrant.d/boxes; ln -s /mnt/.vagrant.d/boxes /root/.vagrant.d/boxes
