FROM golang:1.18-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o signaling-server github.com/PeerCodeProject/SignalingServer/cmd


FROM gcr.io/distroless/base-debian10

WORKDIR /app

COPY --from=build /app/signaling-server /app/signaling-server


ENTRYPOINT [ "./signaling-server" ]