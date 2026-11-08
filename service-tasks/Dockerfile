FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go main.go
COPY internal internal
RUN CGO_ENABLED=0 go build -o /bin/service-tasks main.go

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/service-tasks /service-tasks
ENTRYPOINT ["/service-tasks"]
