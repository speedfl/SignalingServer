FROM golang:1.23-bullseye AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o signaling-server github.com/PeerCodeProject/SignalingServer/cmd

FROM gcr.io/distroless/static-debian11

WORKDIR /app

COPY --from=build /app/signaling-server /app/signaling-server

EXPOSE 4444

ENTRYPOINT [ "./signaling-server" ]