package cachedb_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

var (
	serverBinPath string
	baseURL       string
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "cachedb-test-bin")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	serverBinPath = filepath.Join(tmpDir, "cache-db-test")
	build := exec.Command("go", "build", "-o", serverBinPath, "..")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "build failed:", err)
		os.Exit(1)
	}

	url, stop, err := launchServer(serverBinPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	baseURL = url

	code := m.Run()
	stop()
	os.Exit(code)
}

func freeTCPPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func launchServer(bin string, extraArgs ...string) (string, func(), error) {
	port, err := freeTCPPort()
	if err != nil {
		return "", nil, err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	args := append([]string{"-addr", addr}, extraArgs...)

	cmd := exec.Command(bin, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return "", nil, err
	}

	url := "http://" + addr
	if err := waitHealthy(url, 5*time.Second); err != nil {
		cmd.Process.Kill()
		return "", nil, err
	}

	stop := func() {
		cmd.Process.Kill()
		cmd.Wait()
	}
	return url, stop, nil
}

func waitHealthy(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("server at %s did not become healthy within %s", url, timeout)
}
