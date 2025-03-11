FROM centos:7
WORKDIR /root/
ADD ./server /root/server
ADD ./client/client /root/client/client
ADD ./client/config/config.json /root/client/config/config.json
RUN chmod +x /root/client/client
CMD ["/bin/bash", "-c", "/root/server"]
