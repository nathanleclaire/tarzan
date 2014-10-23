_BIG FAT DISCLAIMER_: This Readme mentions some things which do not exist yet.  `tarzan` is still a work in progress and is not meant to leak to HN etc. yet.

tarzan
======

### Self-hosted Docker Automated Builds that Run Rather Fast

![king of the apes](https://github.com/nathanleclaire/tarzan/blob/master/static/img/tarzan.jpg)

[Automated Builds](http://docs.docker.com/docker-hub/builds/) are one of the best features of the [Docker Hub](https://hub.docker.com).  They allow you to automatically re-create your Docker images on source control push, and they allow other people to find the `Dockerfile` used to create your image to inspect and play around with before pulling, running, or modifying it.

However, there is a big problem incorporating Automated Builds in a real-life workflow in the their current form. 

- Because Docker's build robot runs Automated Builds using the `--no-cache` option (because the infrastructure costs of not doing so would be prohibitive), all of the image layers are created from scratch each time.  
- This ends up in an Automated Build process which could take ten minutes or more (for an operation which would take seconds locally) and does not use the [Docker build cache](http://thenewstack.io/understanding-the-docker-cache-for-faster-builds/) (one of the best, most oft-cited features of `docker build`) at all.
- Because the layers are completely new, Docker's build robot pushes _all new layers_ when it pushes the built image back to Docker Hub, slowing the Automated Build down even more (the familiar `"image layer already exists, skipping"` message is nowhere to be found).
- When end users go to `docker pull` the image built using an Automated Build, they _always_ get new layers even if they have pulled that same image before.  This means that even if you only changed one character in one line of source code, you will most likely have to pull down anywhere from 80 megabytes to a gigabyte or more of Docker image layers.  This makes Automated Builds look very unattractive for real production deployments.

Therefore, it would be highly preferable to have a automated build robot which runs builds using the Docker cache.

`tarzan` is a naive implementation of such an automated build robot, written in [Go](http://golang.org).  I say it is "naive" because it doesn't attempt to do anything particularly clever (largely it shells out to `docker` commands) and is inherently meant to run on a single host (though this may change in the future).  However, it could still be turn out to be a useful tool for automating Docker image re-builds and deploys.

# Getting Started

# FAQ

#### Q: Is this compatible with Docker Hub's automated builds?

Yes and no.  Docker Hub does not allow users to push to automated builds manually (using `docker push`), so it is impossible to use a official Automated Build Docker Hub repository as a backend for your `tarzan` build.  However, nothing is stopping you from creating _two_ Docker Hub repositories (and two separate webhooks on Github) and using one as a normal automated build (allowing for `Dockerfile discovery`) and one as a `tarzan` build (allowing for fast build and pull). 

#### Q: Can I run this using my own registry instead of using Docker Hub as a backend?

Yes.  Provided in this repository is a `fig.yml` file which will allow you to run `fig up` in the project's directory and bootstrap an instance of `tarzan` running alongside a local instance of the [Docker open-source registry](https://github.com/docker/docker-registry) as a backend.  That way, you can also push and pull images from the same host where you are running `tarzan` using the Docker `image.location.com/imagename` format.

#### Q: Why is the project called `tarzan`?

Partially because it is meant to be a wild and feral tool, refusing to be tamed by the confines of civilazation, but there is also a secret meaning and I will buy you a beer, coffee, cookie etc. if you figure it out.
