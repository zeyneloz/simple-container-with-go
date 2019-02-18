package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const (
	// ContainerRootfsPath : Path of the directory which to be used as root filesystem in container.
	// You should have `rootfs` directory which contains a small linux file system.
	// You can use BusyBox or a small Alpine linux image for that.
	// Check https://wiki.alpinelinux.org/wiki/Installing_Alpine_Linux_in_a_chroot
	ContainerRootfsPath = "./rootfs"

	// OldRootPath : Path that old rootfilesystem to be moved in container.
	OldRootPath = ".old_root"

	// ContainerHostName : The name to be used as hostname in container
	ContainerHostName = "container001"
)

func main() {
	switch os.Args[1] {
	case "run":
		selfExec()
	case "spawner":
		spawner()
	default:
		panic("Unknown command")
	}
}

// Set the hostname of container
func setHostName(hostName string) {
	must(syscall.Sethostname([]byte(hostName)))
}

// Make necessary mounts for container such as /proc.
func mounts(newRoot string) {
	procDict := filepath.Join(newRoot, "/proc")
	_ = os.Mkdir(procDict, 0777)
	must(syscall.Mount(newRoot, procDict, "proc", uintptr(0), ""))
}

// Prepare root fielsystem for container.
func prepareRootfs(newRoot string) {
	/*
		From Linux man:
		int pivot_root(const char *new_root, const char *put_old);
			...
			pivot_root() moves the root file system of the calling process to the directory put_old and makes
			new_root the new root file system of the calling process.

			...
				The following restrictions apply to new_root and put_old:
				-  They must be directories.
				-  new_root and put_old must not be on the same filesystem as the current root.
				-  put_old must be underneath new_root, that is, adding a nonzero number
					of /.. to the string pointed to by put_old must yield the same directory as new_root.
				-  No other filesystem may be mounted on put_old.
	*/

	// Since `new_root and put_old must not be on the same filesystem as the current root.`, we need to mount newroot.
	must(syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""))

	// Create `{newRoot}/.oldroot` for old root filesystem.
	putOld := filepath.Join(newRoot, OldRootPath)
	_ = os.Mkdir(putOld, 0777)

	// This is related to the systemd mounts. We should change the roto mount to private before pivotting.
	must(syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""))

	// Move the root filesystem to new filesystem
	must(syscall.PivotRoot(newRoot, putOld))

	// Change working directory to /
	must(syscall.Chdir("/"))

	// Unmount old filesystem which is now under /.old_root
	must(syscall.Unmount(OldRootPath, syscall.MNT_DETACH))

	// Remove temporary old filesystem
	must(os.RemoveAll(OldRootPath))
}

func selfExec() {
	// For some security reasons we should set hostname before running the actual container program. So we need a middle process
	// that is cloned with new namespaces and this process should change hostname, prepare filesystem and then exec the container program.
	cmd := exec.Command("/proc/self/exe", append([]string{"spawner"}, os.Args[2:]...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
}

func spawner() {
	mounts(ContainerRootfsPath)
	prepareRootfs(ContainerRootfsPath)
	setHostName(ContainerHostName)

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
