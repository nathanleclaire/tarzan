from golang:1.3

run apt-get update && apt-get install -y git-core cmake

run git clone --recursive https://github.com/libgit2/git2go /go/src/github.com/libgit2/git2go
run apt-get install -y pkg-config
run cd /go/src/github.com/libgit2/git2go && make install
add ./Godeps/_workspace/src /go/src
add . /go/src/github.com/nathanleclaire/tarzan
workdir /go/src/github.com/nathanleclaire/tarzan

cmd ["go", "build"]
