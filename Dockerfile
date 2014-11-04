from golang:1.3

run apt-get update && apt-get install -y git-core cmake
run apt-get install -y pkg-config


# add vendored deps
add ./Godeps/_workspace/src /go/src

# add src and setup for work on the project
add . /go/src/github.com/nathanleclaire/tarzan
workdir /go/src/github.com/nathanleclaire/tarzan
run go build

# tesat again
# run tarzan binary as non-privileged user in container
run useradd gobuddy
user gobuddy

volume ["/go/src/github.com/nathanleclaire/tarzan/repos"]

entrypoint ["./tarzan"]
