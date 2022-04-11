package main

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	chattrCmd = "chattr"
	lsattrCmd = "lsattr"
)

func AddImmutable(path string) bool {
	chattrBin := which(chattrCmd)
	if _, err := os.Stat(path); err == nil {
		cmd := exec.Command(chattrBin, "+i", path)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err = cmd.Run(); err != nil {
			return false
			// return fmt.Errorf("%s +i failed: %s. Err: %v", chattrBin, stderr.String(), err)
		}
	}

	return true
}

func RemoveImmutable(path string) bool {
	chattrBin := which(chattrCmd)
	if _, err := os.Stat(path); err == nil {
		cmd := exec.Command(chattrBin, "-i", path)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err = cmd.Run(); err != nil {
			return false
			// return fmt.Errorf("%s -i failed: %s. Err: %v", chattrBin, stderr.String(), err)
		}
	}

	return true
}

func IsImmutable(path string) bool {
	lsattrBin := which(lsattrCmd)
	if _, err := os.Stat(path); err != nil {
		// fmt.Errorf("Failed to stat mount path:%v", err)
		return true
	}
	op, err := exec.Command(lsattrBin, "-d", path).CombinedOutput()
	if err != nil {
		// Cannot get path status, return true so that immutable bit is not reverted
		// fmt.Errorf("Error listing attrs for %v err:%v", path, string(op))
		return true
	}
	// 'lsattr -d' output is a single line with 2 fields separated by space; 1st one
	// is list of applicable attrs and 2nd field is the path itself.Sample output below.
	// lsattr -d /mnt/vol2
	// ----i--------e-- /mnt/vol2
	attrs := strings.Split(string(op), " ")
	if len(attrs) != 2 {
		// Cannot get path status, return true so that immutable bit is not reverted
		// fmt.Errorf("Invalid lsattr output %v", string(op))
		return true
	}
	if strings.Contains(attrs[0], "i") {
		// fmt.Println("Path %v already set to immutable", path)
		return true
	}

	return false
}

func which(bin string) string {
	pathList := []string{"/usr/bin", "/sbin", "/usr/sbin", "/usr/local/bin"}
	for _, p := range pathList {
		if _, err := os.Stat(path.Join(p, bin)); err == nil {
			return path.Join(p, bin)
		}
	}
	return bin
}
