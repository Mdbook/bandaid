package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"io/ioutil"
	"os"
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
		location.Checksum, err = location.GetSHA()
	}
	if err {
		Warnf("Filepath error while importing %s. Skipping...\n", a.Name)
		return false
	}
	return true
}
func (a *ServiceObject) InitSO() bool {
	var err bool
	a.Checksum, err = a.GetSHA()
	if err {
		Warnf("Filepath error while importing %s. Skipping...\n", a.Name)
		return false
	}
	return true
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

func (a *ServiceObject) InitBackup() {
	f, _ := os.Open(a.Path)
	a.Backup, _ = ioutil.ReadAll(f)
	defer f.Close()
	stat, _ := os.Stat(a.Path)
	a.Mode = stat.Mode()
}
