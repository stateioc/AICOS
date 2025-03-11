FROM centos:7
RUN yum install -y ethtool && yum install -y pciutils
ADD ./node-reporter /root/node-reporter
CMD ["/bin/bash", "-c", "/root/node-reporter"]