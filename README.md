# Very simple container using golang   
This is a very basic example of how docker creates the containers using linux namespaces under the hood.
- You need to run this go program on a Linux machine. I suggest working on a virtual machine.
- `rootfs` is just a small linux root file system. I used Busybox rootfs

## TL;DR
- Clone git repo
- run `mkdir rootfs`
- run `tar xvf rootfs.tar -C rootfs`
- Now you are ready to run the container: `go run engine.go run /bin/sh`
- then you will have a ssh connection in to the container.