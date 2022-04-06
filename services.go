package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"io/ioutil"
	"os"
	"strings"
)

type ServiceObject struct {
	Name     string
	Mode     fs.FileMode
	Path     string
	Checksum string
	Backup   []byte
}

type Service struct {
	Name      string `json:"name"`
	locations []*ServiceObject
	Binary    *ServiceObject `json:"binary"`
	Service   *ServiceObject `json:"service"`
	Config    *ServiceObject `json:"config"`
}
type Services struct {
	Services []Service       `json:"services"`
	Files    []ServiceObject `json:"other_files"`
}

func (a *ServiceObject) CheckSHA() bool {
	sha, err := a.GetSHA()
	if err {
		return false
	}
	if sha != a.Checksum {
		return false
	}
	return true
}

func (a *Service) Init() bool {
	var err bool
	a.locations = []*ServiceObject{
		a.Binary,
		a.Service,
		a.Config,
	}
	for _, location := range a.locations {
		if !FileExists(location.Path) {
			Warnf("Filepath error while importing %s. Skipping...\n", a.Name)
			return false
		}
		location.Checksum, err = location.GetBackupSHA()
	}
	if err {
		Warnf("Filepath error while importing %s. Skipping...\n", a.Name)
		return false
	}
	return true
}

func (a *ServiceObject) InitSO() bool {
	var err bool
	a.Checksum, err = a.GetBackupSHA()
	if err {
		Warnf("Filepath error while importing %s. Skipping...\n", a.Name)
		return false
	}
	return true
}

func (a *ServiceObject) GetBackupSHA() (string, bool) {
	var path string = a.Path
	filename := GetConfigName(a.Path)
	if FileExists(filename) && loadFromConfig {
		path = filename
	}
	f, err := os.Open(path)
	if err != nil {
		return "ERR", true
	}
	defer f.Close()
	read, err := ioutil.ReadAll(f)
	sha := sha256.Sum256(read)
	ret := hex.EncodeToString(sha[:])
	return ret, false
}

func (a *ServiceObject) GetSHA() (string, bool) {
	f, err := os.Open(a.Path)
	if err != nil {
		return "ERR", true
	}
	defer f.Close()
	read, err := ioutil.ReadAll(f)
	sha := sha256.Sum256(read)
	ret := hex.EncodeToString(sha[:])
	return ret, false
}

func GetConfigName(s string) string {
	var filename string = s
	if strings.Contains(s, "\\") {
		filename = strings.Join(strings.Split(s, "\\"), "/")
	}
	filename = strings.Join(strings.Split(filename, "/"), "._.")
	return ".bandaid/" + filename
}

func (a *ServiceObject) InitBackup() {
	filename := GetConfigName(a.Path)
	var path string = a.Path
	if FileExists(filename) && loadFromConfig {
		path = filename
	}
	f, _ := os.Open(path)
	a.Backup, _ = ioutil.ReadAll(f)
	defer f.Close()
	stat, _ := os.Stat(path)
	a.Mode = stat.Mode()
	cnfPath := GetConfigName(a.Path)
	writeFile(cnfPath, a.Backup)
	os.Chmod(cnfPath, a.Mode)
}

func InitConfigFolder() {
	if FileExists(".bandaid") {
		Warnf("Detected .bandaid folder. Load from backup? [y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		rawAnswer, _ := reader.ReadString('\n')
		answer := trim(rawAnswer)
		if answer == "y" {
			loadFromConfig = true
		} else {
			loadFromConfig = false
		}
		return
	}
	err := os.Mkdir(".bandaid", 0755)
	if err != nil {
		Errorf("Could not create config directory\n")
	}
}
