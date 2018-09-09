FROM centos/systemd
RUN yum install -y epel-release
RUN yum install -y tito golang make
COPY . /src
WORKDIR /src
RUN make rpm
