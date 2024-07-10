FROM ubuntu

# install Ansible
COPY ./ansible /ansible
COPY ./downloads /ansible/downloads
RUN apt update; \
    apt install -y ansible; \
    apt install -y git; \
    cd /ansible ; \
    ansible-galaxy install -r requirements.yml; \
    ansible-playbook --connection=local -i inventory.localhost.yml vagrant.yml; \
    rm -rf /ansible; \
    rm -rf /root/.ansible ; \
    apt-get clean

RUN . /etc/profile.d/golang.sh; \
    mkdir -p $GOPATH/src/; \
    cd $GOPATH/src/; \
    git clone git@github.com:resurgentech/vagrant_tools.git; \
    cd $GOPATH/src/vagrant_tools/; \
    go get; \
    export GIT_COMMIT=$(git rev-list -1 HEAD) && \
    go build -ldflags "-X main.GitCommit=$GIT_COMMIT"; \
    mv vagrant_tools /bin; \
    cd $GOPATH/src/; \
    rm -rf vagrant_tools;


RUN rm -rf /root/.vagrant.d/boxes/; mkdir -p /mnt/.vagrant.d/boxes; ln -s /mnt/.vagrant.d/boxes /root/.vagrant.d/boxes
