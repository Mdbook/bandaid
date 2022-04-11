package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"syscall"
)

type ServiceObject struct {
	Mode     fs.FileMode
	Name     string
	Owner    int
	Group    int
	Path     string
	Checksum string
	Backup   []byte
	isDir    bool
}

type Directory struct {
	Name        string `json:"name"`
	Path        string
	isRecursive bool
	files       []*ServiceObject
}

type Service struct {
	Name      string `json:"name"`
	locations []*ServiceObject
	Binary    *ServiceObject `json:"binary"`
	Service   *ServiceObject `json:"service"`
	Config    *ServiceObject `json:"config"`
}
type Services struct {
	Services    []Service       `json:"services"`
	Files       []ServiceObject `json:"other_files"`
	Directories []Directory     `json:"directories"`
}

func (a *ServiceObject) CheckPerms() bool {
	stat, _ := os.Stat(a.Path)
	if stat.Mode() != a.Mode {
		if config.outputEnabled {
			fmt.Printf("\nPermissions for %s have been modified. Restoring...\n", a.Name)
		}
		return false
	}
	inf := stat.Sys().(*syscall.Stat_t)
	if int(inf.Uid) != a.Owner || int(inf.Gid) != a.Group {
		if config.outputEnabled {
			fmt.Printf("Permissions for %s have been modified. Restoring...\n", a.Name)
		}
		return false
	}
	return true
}

func (a *ServiceObject) CheckFile() bool {
	if a.isDir {
		if !FileExists(a.Path) {
			return false
		}
		return true
	}
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

func AddDir(path string, files []*ServiceObject) []*ServiceObject {
	items, _ := ioutil.ReadDir(path)
	for _, item := range items {
		subPath := ConcatenatePath(path, item.Name())
		if item.IsDir() {
			newDir := &ServiceObject{
				Name:  subPath,
				Path:  subPath,
				isDir: true,
			}
			if newDir.InitSO() {
				newDir.InitBackup()
				files = append(files, newDir)
				files = append(files, AddDir(subPath, []*ServiceObject{})...)
			}
		} else {
			newFile := &ServiceObject{
				Name:  subPath,
				Path:  subPath,
				isDir: false,
			}
			if newFile.InitSO() {
				newFile.InitBackup()
				files = append(files, newFile)
			}
		}
	}
	return files
}

func (a *Directory) InitDir() bool {
	if FileExists(a.Path) {
		topDir := &ServiceObject{
			Name:  a.Path,
			Path:  a.Path,
			isDir: true,
		}
		if topDir.InitSO() {
			topDir.InitBackup()
		}
		a.files = AddDir(a.Path, []*ServiceObject{topDir})
		return true
	}
	Warnf("Directory %s doesn't exist. Skipping...", a.Path)
	return false
}

func (a *ServiceObject) GetBackupSHA() (string, bool) {
	var path string = a.Path
	filename := GetConfigName(a.Path)
	doEncrypt := false
	if FileExists(filename) && config.loadFromConfig {
		path = filename
		doEncrypt = true
	}
	f, err := os.Open(path)
	if err != nil {
		return "ERR", true
	}
	defer f.Close()
	read, err := ioutil.ReadAll(f)
	if config.doEncryption && doEncrypt {
		read = decrypt(read, config.key)
	}
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

func (a *ServiceObject) FreeBackup() {
	filename := GetConfigName(a.Path)
	if FileExists(filename) {
		os.Remove(filename)
	}
}

func (a *ServiceObject) InitBackup() {
	filename := GetConfigName(a.Path)
	var path string = a.Path
	doEncrypt := false
	if FileExists(filename) && config.loadFromConfig {
		doEncrypt = true
		path = filename
	}
	stat, _ := os.Stat(path)
	inf := stat.Sys().(*syscall.Stat_t)
	a.Owner = int(inf.Uid)
	a.Group = int(inf.Gid)
	a.Mode = stat.Mode()
	if !a.isDir {
		f, _ := os.Open(path)
		a.Backup, _ = ioutil.ReadAll(f)
		if doEncrypt {
			a.Backup = decrypt(a.Backup, config.key)
		}
		defer f.Close()
	}
	if config.doBackup {
		cnfPath := GetConfigName(a.Path)
		if a.isDir {
			cnfPath = cnfPath + "._."
		} else if config.doEncryption {
			writeFile(cnfPath, encrypt(a.Backup, config.key))
		} else {
			writeFile(cnfPath, a.Backup)
		}
		os.Chmod(cnfPath, a.Mode)
		os.Chown(cnfPath, a.Owner, a.Group)
	}
}

func InitConfigFolder() {
	if config.doEncryption {
		Warnf("Please input the encryption/decryption key to use: ")
		key := GetInput()
		config.key = GetPass(key)
	}
	if FileExists(config.backupLocation) && config.loadFromConfig {
		Warnf("Detected backup folder (%s). Load from backup? [y/n]: ", config.backupLocation)
		if GetInput() == "y" {
			config.loadFromConfig = true
		} else {
			config.loadFromConfig = false
		}
	} else if config.doBackup && config.loadFromConfig {
		err := os.Mkdir(config.backupLocation, 0755)
		if err != nil {
			Errorf("Could not create config directory (%s)\n", config.backupLocation)
		}
	}
}

func (e *ServiceObject) writeBackup() bool {
	if e.isDir {
		if !FileExists(e.Path) {
			err := os.Mkdir(e.Path, e.Mode)
			if err != nil {
				if config.outputEnabled {
					Warnf("Error: Could not restore directory %s", e.Name)
				}
				return false
			}
		}
		os.Chmod(e.Path, e.Mode)
		os.Chown(e.Path, e.Owner, e.Group)
		return true
	}
	if !FileExists(e.Path) {
		if config.outputEnabled {
			fmt.Printf("File %s was deleted. Restoring...\n", e.Path)
		}
	} else if IsImmutable(e.Path) {
		if config.outputEnabled {
			fmt.Printf("File %s is immutable. Removing immutable flag...\n", e.Path)
		}
		RemoveImmutable(e.Path)
	}
	ret := writeFile(e.Path, e.Backup)
	if ret {
		err := os.Chmod(e.Path, e.Mode)
		if err != nil {
			if config.outputEnabled {
				fmt.Printf("Error setting permissions for %s", e.Path)
			}
			return false
		}
		os.Chown(e.Path, e.Owner, e.Group)
	}
	return ret
}

func (e *ServiceObject) WritePerms() bool {
	err := os.Chmod(e.Path, e.Mode)
	if err != nil {
		return false
	}
	err = os.Chown(e.Path, e.Owner, e.Group)
	if err != nil {
		return false
	}
	return true
}
