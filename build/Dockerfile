FROM golang:1.20.1

RUN apt-get update \
  && apt-get -y --no-install-recommends install \
    crossbuild-essential-arm64 \
    crossbuild-essential-armel \
    crossbuild-essential-armhf \
    crossbuild-essential-ppc64el

WORKDIR /usr/local/src/dex

COPY go.mod go.sum ./
COPY api/v2/go.mod api/v2/go.sum ./api/v2/
RUN go mod download

COPY . .

CMD ["go", "run", "build/build.go"]
