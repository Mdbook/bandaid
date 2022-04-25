/*
services.go- Contains mostly everything to do with
service/file/directory objects
*/

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

// File object
type ServiceObject struct {
	Mode     fs.FileMode // File permissions
	Name     string
	Owner    int // UID
	Group    int // GID
	Path     string
	Checksum string
	Backup   []byte // Contents of the file are stored in memory
	isDir    bool
}

type Directory struct {
	Name        string `json:"name"` // These tags are added let us use JSON unmarshal
	Path        string
	isRecursive bool
	files       []*ServiceObject // Store pointers instead of actual variables to aid with making changes
}

type Service struct {
	Name      string           `json:"name"`
	locations []*ServiceObject // We store the objects in this array as well, to allow for easier iterating
	Binary    *ServiceObject   `json:"binary"`
	Service   *ServiceObject   `json:"service"`
	Config    *ServiceObject   `json:"config"`
}

type Services struct {
	Services    []Service       `json:"services"`
	Files       []ServiceObject `json:"other_files"`
	Directories []Directory     `json:"directories"`
}

// Check to see if the permissions for a file have been modified
func (a *ServiceObject) CheckPerms() bool {
	stat, _ := os.Stat(a.Path)
	if stat.Mode() != a.Mode {
		if config.outputEnabled {
			fmt.Printf("\nPermissions for %s have been modified. Restoring...\n", a.Name)
		}
		return false
	}
	// Also check uid and gid
	inf := stat.Sys().(*syscall.Stat_t)
	if int(inf.Uid) != a.Owner || int(inf.Gid) != a.Group {
		if config.outputEnabled {
			fmt.Printf("Permissions for %s have been modified. Restoring...\n", a.Name)
		}
		return false
	}
	return true
}

// Check to see if file has been deleted or modified
func (a *ServiceObject) CheckFile() bool {
	if a.isDir {
		return FileExists(a.Path)
	}
	// Get the SHA checksum of the file's current state
	// and compare it to the one stored in memory
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
	// Initialize the locations array
	a.locations = []*ServiceObject{
		a.Binary,
		a.Service,
		a.Config,
	}
	for _, location := range a.locations {
		// Make sure that the file exists
		// TODO should I delete this?
		if !FileExists(location.Path) {
			Warnf("Filepath error while importing %s. Skipping...\n", a.Name)
			return false
		}
		// If it does, get the SHA (from backup or from current state if no backup / disabled)
		location.Checksum, err = location.GetBackupSHA()
	}
	if err {
		Warnf("Filepath error while importing %s. Skipping...\n", a.Name)
		return false
	}
	return true
}

// Initialize a file object
func (a *ServiceObject) InitSO() bool {
	var err bool
	a.Checksum, err = a.GetBackupSHA()
	if err {
		Warnf("Filepath error while importing %s. Skipping...\n", a.Name)
		return false
	}
	return true
}

// Add a directory and all of its files & subfolders recursively
func AddDir(path string, files []*ServiceObject) []*ServiceObject {
	// Iterate through all items in the directory
	items, _ := ioutil.ReadDir(path)
	for _, item := range items {
		// Get the path of the current item
		subPath := ConcatenatePath(path, item.Name())
		if item.IsDir() {
			// Create a file object for the directory
			newDir := &ServiceObject{
				Name:  subPath,
				Path:  subPath,
				isDir: true,
			}
			// Make sure directory successfully inits first
			if newDir.InitSO() {
				newDir.InitBackup()
				files = append(files, newDir)
				// Recusively add all files and items in the subdirectory
				files = append(files, AddDir(subPath, []*ServiceObject{})...)
			}
		} else {
			// If the item is a file, just add it to the files array
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

// Initialize a directory object
func (a *Directory) InitDir() bool {
	if FileExists(a.Path) {
		// Create the ServiceObject for the top directory
		topDir := &ServiceObject{
			Name:  a.Path,
			Path:  a.Path,
			isDir: true,
		}
		// Make sure the top directory inits
		if topDir.InitSO() {
			topDir.InitBackup()
		}
		// Add all files recursively
		a.files = AddDir(a.Path, []*ServiceObject{topDir})
		return true
	}
	Warnf("Directory %s doesn't exist. Skipping...", a.Path)
	return false
}

// Get the SHA-256 checksum for a file,
// either from encrypted backup (if enabled)
// or from the original file
func (a *ServiceObject) GetBackupSHA() (string, bool) {
	// Set local path variable
	var path string = a.Path
	// Grab the path for the backup folder
	filename := GetConfigName(a.Path)
	doEncrypt := false
	// Check if the backup path exists
	if FileExists(filename) && config.loadFromConfig {
		// If it does, update the path and set to decrypt
		path = filename
		doEncrypt = true
	}
	f, err := os.Open(path)
	if err != nil {
		return "ERR", true
	}
	defer f.Close()
	read, err := ioutil.ReadAll(f)
	if err != nil {
		return "ERR", true
	}
	// If we're reading from a backup, decrypt it
	if config.doEncryption && doEncrypt {
		read = decrypt(read, config.key)
	}
	// Return the sha256 in string format
	sha := sha256.Sum256(read)
	ret := hex.EncodeToString(sha[:])
	return ret, false
}

// Get the SHA256 checksum of a file object
func (a *ServiceObject) GetSHA() (string, bool) {
	f, err := os.Open(a.Path)
	if err != nil {
		return "ERR", true
	}
	defer f.Close()
	read, _ := ioutil.ReadAll(f)
	sha := sha256.Sum256(read)
	ret := hex.EncodeToString(sha[:])
	return ret, false
}

// Remove the backup file when freeing a file
func (a *ServiceObject) FreeBackup() {
	filename := GetConfigName(a.Path)
	if FileExists(filename) {
		os.Remove(filename)
	}
}

// Initialize the backup for a file
func (a *ServiceObject) InitBackup() {
	// Get the file backup path
	filename := GetConfigName(a.Path)
	var path string = a.Path
	doEncrypt := false
	// Check to see if we're loading from a backup
	if FileExists(filename) && config.loadFromConfig {
		// If so, update the path
		doEncrypt = true
		path = filename
	}
	// Get permissions, owner, etc.
	stat, _ := os.Stat(path)
	inf := stat.Sys().(*syscall.Stat_t)
	a.Owner = int(inf.Uid)
	a.Group = int(inf.Gid)
	a.Mode = stat.Mode()
	// If the file isn't a directory, read and store the file's contents
	if !a.isDir {
		f, _ := os.Open(path)
		a.Backup, _ = ioutil.ReadAll(f)
		if doEncrypt {
			a.Backup = decrypt(a.Backup, config.key)
		}
		defer f.Close()
	}
	if config.doBackup {
		// Write to the backup file
		cnfPath := GetConfigName(a.Path)
		if a.isDir {
			cnfPath = cnfPath + "._."
		} else if config.doEncryption {
			writeFile(cnfPath, encrypt(a.Backup, config.key))
		} else {
			writeFile(cnfPath, a.Backup)
		}
		// Set the backup's permisssions to match the file
		os.Chmod(cnfPath, a.Mode)
		os.Chown(cnfPath, a.Owner, a.Group)
	}
}

// Initialize the backup folder
func InitConfigFolder() {
	// Grab the encryption key from the user
	if config.doEncryption {
		Warnf("Please input the encryption/decryption key to use: ")
		key := GetInput()
		// Make it a key and convert it into a bytes[] object
		config.key = GetPass(key)
	}
	// Set loadFromConfig
	if FileExists(config.backupLocation) && config.loadFromConfig {
		Warnf("Detected backup folder (%s). Load from backup? [y/n]: ", config.backupLocation)
		if GetInput() == "y" {
			config.loadFromConfig = true
		} else {
			config.loadFromConfig = false
		}
	} else if config.doBackup && config.loadFromConfig {
		// If backup directory doesn't exist, create it
		err := os.Mkdir(config.backupLocation, 0755)
		if err != nil {
			Errorf("Could not create config directory (%s)\n", config.backupLocation)
		}
	}
}

// Restore the backup of a file/folder
func (e *ServiceObject) writeBackup() bool {
	if e.isDir {
		// Each file object is scanned directly, so once we restore
		// the directory, the files and directories below it will
		// automatically be restored as well
		if !FileExists(e.Path) {
			err := os.Mkdir(e.Path, e.Mode)
			if err != nil {
				if config.outputEnabled {
					Warnf("Error: Could not restore directory %s", e.Name)
				}
				return false
			}
		}
		// We don't really need this but better safe than sorry idk
		os.Chmod(e.Path, e.Mode)
		os.Chown(e.Path, e.Owner, e.Group)
		return true
	}
	// Check to see if the file was deleted or just modified
	if !FileExists(e.Path) {
		if config.outputEnabled {
			fmt.Printf("File %s was deleted. Restoring...\n", e.Path)
		}
		// See if file is immutable
	} else if IsImmutable(e.Path) {
		if config.outputEnabled {
			fmt.Printf("File %s is immutable. Removing immutable flag...\n", e.Path)
		}
		RemoveImmutable(e.Path)
	}
	// Restore the backup
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

// Revert the permissions of a file back to its backup
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
