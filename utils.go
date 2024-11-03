package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
)

// func pp(args ...interface{}) {
// 	fmt.Printf("%v\n", args)
// }

func qemuArch() string {
	arch := runtime.GOARCH
	if arch == "arm64" {
		arch = "aarch64"
	}
	if arch == "amd64" {
		arch = "x86_64"
	}
	return arch
}

func checkRequirements() {
	arch := qemuArch()

	reqs := []struct {
		k string
		t int
	}{
		{k: fmt.Sprintf("qemu-system-%s", arch), t: 1},
		{k: "qemu-img", t: 1},
		{k: "hdiutil", t: 1},
		{k: "route", t: 1},
		{k: "awk", t: 1},
		{k: fmt.Sprintf("/opt/local/share/qemu/edk2-%s-code.fd", arch), t: 0},
	}
	for _, r := range reqs {
		if r.t == 0 && !file_exists(r.k) {
			fatalf("Missing file %s", r.k)
		}
		if r.t == 1 {
			if path, _ := exec.LookPath(r.k); len(path) == 0 {
				fatalf("Executable %s not found in PATH", r.k)
			}
		}
	}
}

func genMACAddr() (string, error) {
	mac := make([]byte, 6)
	_, err := rand.Read(mac)
	if err != nil {
		return "", err
	}

	// Set the local bit and clear the multicast bit for the first byte
	mac[0] = (mac[0] | 0x02) & 0xfe

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]), nil
}

func IPv4ToIPv6(ipv4Str string) (string, error) {
	ipv4 := net.ParseIP(ipv4Str)
	if ipv4 == nil {
		return "", fmt.Errorf("invalid IPv4 address: %s", ipv4Str)
	}

	ipv6 := net.IPv6zero
	ipv6 = ipv6.To16() // Ensure it is 128 bits
	ipv4 = ipv4.To4()  // Ensure it is 32 bits

	copy(ipv6[12:], ipv4)

	return ipv6.String(), nil
}

// ============================================================================
// File utilities
// ============================================================================

func file_exists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func file_cp(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
