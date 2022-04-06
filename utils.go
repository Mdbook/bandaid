package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
)

type Colors struct {
	red     string
	green   string
	blue    string
	black   string
	yellow  string
	magenta string
	cyan    string
	white   string
	reset   string
}

func InitColors() Colors {
	if runtime.GOOS == "windows" {
		return Colors{
			reset:   "",
			black:   "",
			red:     "",
			green:   "",
			yellow:  "",
			blue:    "",
			magenta: "",
			cyan:    "",
			white:   "",
		}
	} else {
		return Colors{
			reset:   "\033[0m",
			black:   "\033[30m",
			red:     "\033[31m",
			green:   "\033[32m",
			yellow:  "\033[33m",
			blue:    "\033[34m",
			magenta: "\033[35m",
			cyan:    "\033[36m",
			white:   "\033[37m",
		}
	}

}

func CopyFile(src, dst string) (int64, error) {
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

func removeService(slice []Service, s int) []Service {
	if s == len(slice) {
		return slice[:s-1]
	} else {
		return append(slice[:s], slice[s+1:]...)
	}
}
func removeSO(slice []ServiceObject, s int) []ServiceObject {
	if s == len(slice) {
		return slice[:s-1]
	} else {
		return append(slice[:s], slice[s+1:]...)
	}
}

func (e *Service) getAttr(field string) *ServiceObject {
	return e.locations[find(serviceNames, field)]
}

func contains(arr []string, s string) bool {
	for _, str := range arr {
		if str == s {
			return true
		}
	}
	return false
}

func containsInt(arr []int, i int) bool {
	for _, e := range arr {
		if e == i {
			return true
		}
	}
	return false
}

func find(arr []string, s string) int {
	for i, str := range arr {
		if str == s {
			return i
		}
	}
	return -1
}

func (e *ServiceObject) writeBackup() bool {
	if !FileExists(e.Path) {
		if outputEnabled {
			fmt.Printf("File %s was deleted. Restoring...\n", e.Path)
		}
	} else if IsImmutable(e.Path) {
		if outputEnabled {
			fmt.Printf("File %s is immutable. Removing immutable flag...\n", e.Path)
		}
		RemoveImmutable(e.Path)
	}
	ret := writeFile(e.Path, e.Backup)
	if ret {
		err := os.Chmod(e.Path, e.Mode)
		if err != nil {
			if outputEnabled {
				fmt.Printf("Error setting permissions for %s", e.Path)
			}
			return false
		}
	}
	return ret
}

func writeFile(file string, contents []byte) bool {
	f, err := os.Create(file)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Write(contents)
	if err != nil {
		return false
	}
	return true
}

func readFile(path string) string {
	dat, _ := ioutil.ReadFile(path)
	str := string(dat)
	return str
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func GetTail(str string, separator string) string {
	s := strings.Split(str, separator)
	return s[len(s)-1]
}

func trim(str string) string {
	return strings.TrimSuffix(strings.TrimSuffix(str, "\n"), "\r")
}

func caret() {
	fmt.Print(colors.green + "> " + colors.reset)
}

func Warnf(s string, params ...interface{}) {
	fmt.Printf(colors.yellow+s+colors.reset, params...)
}

func Errorf(s string, params ...interface{}) {
	fmt.Printf(colors.red+s+colors.reset, params...)
}
