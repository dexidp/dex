FROM centos/systemd
RUN yum install -y epel-release
RUN yum install -y tito golang make
COPY . /src
WORKDIR /src
RUN make rpm
FROM centos/systemd
COPY --from=0 /tmp/tito/x86_64/*rpm /tmp/
RUN rpm -Uv /tmp/*rpm