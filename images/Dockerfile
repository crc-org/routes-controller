FROM quay.io/centos/centos:stream9 AS build
WORKDIR $APP_ROOT/src
RUN yum -y install golang
COPY . .
RUN make

FROM quay.io/centos/centos:stream9
COPY --from=build /src/routes-controller .
ENTRYPOINT ["/routes-controller"]
