package test_with_docker

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)

type LocalDaemon struct {
	serviceTimeout time.Duration
}

func (ld *LocalDaemon) ComposeServices(dir string, svcs serviceMap) (*command, error) {
	ip, _ := ld.IP()

	return composeService(dir, ip, []string{}, svcs, ld.serviceTimeout)
}

// InstallFile puts a path found on the local machine to a path on the docker host.
func (ld *LocalDaemon) InstallFile(src string, dest string) error {
	destDir := filepath.Dir(dest)
	ld.Exec("mkdir", "-p", destDir)
	return ld.Exec("cp", src, dest)
}

// DifferingFiles compares specific files involved in docker
func (ld *LocalDaemon) DifferingFiles(pathPairs ...[]string) (differentPairs [][]string, err error) {
	localPaths, remotePaths := make([]string, 0, len(pathPairs)), make([]string, 0, len(pathPairs))

	for _, pair := range pathPairs {
		localPaths = append(localPaths, pair[0])
		remotePaths = append(remotePaths, pair[1])
	}

	localMD5s := localMD5s(localPaths...)
	remoteMD5s, err := ld.MD5s(remotePaths...)
	if err != nil {
		return
	}

	return fileDiffs(pathPairs, localMD5s, remoteMD5s), nil
}

// IP returns the IP address where the daemon is located.
// In order to access the services provided by a docker-compose on a
// docker-machine, we need to know the ip address. Some client test code
// needs to know the IP address prior to starting up the services, which is
// why this function is exposed
func (ld *LocalDaemon) IP() (net.IP, error) {
	return net.ParseIP(`127.0.0.1`), nil
}

// MD5s computes digests of a list of paths
// This can be used to compare to local digests and avoid copying files or
// restarting the daemon
func (ld *LocalDaemon) MD5s(paths ...string) (map[string]string, error) {
	dir, err := ioutil.TempDir("/tmp", "md5")
	if err != nil {
		return nil, err
	}
	tgts := make([]string)
	md5 := make(map[string]string)
	for _, p := range paths {
		tgt := filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(tgt), os.ModeDir|os.ModePerm)
		ld.Exec("cp", p, tgt)
		tgts = append(tgts, tgt)
		md5[p] = tgt
	}
	ld.Exec("chmod", "-R", "gwo+rw", dir)
	tmd5s := localMD5s(tgts...)
	os.RemoveAll(dir)
	for p, t := range md5 {
		md5[p] = tmd5s[t]
	}
	return md5, nil
}

// RebuildService forces the rebuild of a docker-compose service.
func (ld *LocalDaemon) RebuildService(dir, name string) error {
	return rebuildService(dir, name, []string{})
}

// Shutdown terminates the set of services started by ComposeServices
// If passed a nil (as ComposeServices returns in the event that all services
// were available, Shutdown is a no-op
func (ld *LocalDaemon) Shutdown(c *command) {
	if c != nil {
		dockerComposeDown(c)
	}
}

// RestartDaemon reboots the docker daemon
func (ld *LocalDaemon) RestartDaemon() error {
	rss := [][]string{
		[]string{"/etc/init.d/docker", "restart"},
		[]string{"service", "docker", "restart"},
		[]string{"systemctl", "restart", "docker", "docker.socket"},
	}
	var err error

	for _, rs := range rss {
		err = ld.Exec(rs...)
		if err == nil {
			return err
		}
	}
	return err
}

// Exec executes commands as root on the daemon host
// It uses sudo
func (ld *LocalDaemon) Exec(args ...string) error {
	cmd := runCommand("sudo", args...)
	log.Println(cmd.String())
	return cmd.err
}

/*
 */
