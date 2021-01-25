FROM registry.access.redhat.com/ubi8/ubi AS build
WORKDIR /go/src/app
RUN yum -y install golang make
COPY . .
RUN make

FROM scratch
COPY --from=build /go/src/app/routes-controller .
ENTRYPOINT ["/routes-controller"]
