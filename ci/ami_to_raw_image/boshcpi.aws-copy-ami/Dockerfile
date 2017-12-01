FROM boshcpi/light-stemcell-builder

RUN wget -O /tmp/vagrant.deb https://releases.hashicorp.com/vagrant/1.8.1/vagrant_1.8.1_x86_64.deb && \
    dpkg -i /tmp/vagrant.deb && \
    vagrant plugin install vagrant-aws && \
    rm /tmp/vagrant.deb
