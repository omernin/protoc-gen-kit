VERSION 0.6

copy-code:
    FROM golang:1.18-buster
    WORKDIR /code
    COPY --dir go.mod go.sum ./
    COPY --dir */. *go* ./
    SAVE ARTIFACT /code

compile:
    FROM +copy-code
    RUN go build
    RUN mv protoc-gen-kit /protoc-gen-kit
    SAVE ARTIFACT /protoc-gen-kit

build-image: 
    # Note - we wrap the "namely/protoc-all" image as it already has all the protoc apps and
    # needed dependencies installed.
    # https://hub.docker.com/r/namely/protoc-all
    # https://github.com/namely/docker-protoc
    FROM namely/protoc-all
    # Note - the "namely/protoc-all" save all proto binaries here
    COPY +compile/protoc-gen-kit /usr/local/bin/
    # Note - we rewrite the entrypoint.sh file in order to add the "with-kit" flag
    COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
    SAVE IMAGE --push omernin/protoc-all-with-kit