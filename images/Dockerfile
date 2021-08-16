FROM registry.access.redhat.com/ubi8/go-toolset:1.15.14 AS build
WORKDIR $APP_ROOT/src
COPY . .
RUN make

FROM registry.access.redhat.com/ubi8/ubi
COPY --from=build /opt/app-root/src/routes-controller .
ENTRYPOINT ["/routes-controller"]
