/*
Bandaid: Made by Mikayla Burke
Routinely checks all files and folders added in config
for modifications, and reverts them to their previous
state if they have been modified in any way.
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Initialize global variables
// var services []*Service
var master Services
var serviceNames []string = []string{
	"Binary",
	"Service",
	"Config",
}
var isFreeing sync.Mutex
var colors Colors = InitColors()
var config Config

func main() {
	/*
		For linux, we can use syscall to detect if
		user is root. This is no longer needed
		because of the creation of /dev/nil at
		the CreateNil() function.
			err := syscall.Setuid(0)
			if err != nil {
				Errorf("Error: must be run as root\n")
				os.Exit(-1)
			}
	*/
	HandleArgs()
	CreateNil()
	InitConfigFolder()
	// TODO encrypt the files stored in memory as well
	master = InitConfig()
	InitBackups()
	fmt.Println()
	PrintChecksums()
	Warnf("\nIpChairs is disabled by default. Run the ipchairs command to configure.\n")
	fmt.Printf("\n%sBandaid is active.%s\n", colors.yellow, colors.reset)
	// Start the main process
	go RunBandaid()
	// Fixing ICMP is its own function since it has its own delay
	go FixICMP()
	// Initialize the IpChairs object and run it
	InitIpChairs()
	go ipchairs.Start()
	InputCommand()
}

// Function to establish config and handle user arguments
func HandleArgs() {
	// Initialize default config
	config = Config{
		delay:           1000,
		icmpDelay:       10,
		configFile:      "config.json",
		backupLocation:  ".bandaid",
		key:             GetPass("changeme"),
		outputEnabled:   true,
		loadFromConfig:  true,
		upkeep:          true,
		doBackup:        true,
		checkPerms:      true,
		doEncryption:    true,
		ipChairs:        true,
		ipChairsConsole: false,
	}
	// If there are no command line arguments, we return after setting the default config
	if len(os.Args) <= 1 {
		return
	}
	for i, arg := range os.Args[1:] {
		switch arg {
		case "-h", "--help":
			fmt.Printf( //TODO add optional encryption
				colors.green + "Bandaid v1.3: Made by Mikayla Burke\n" + colors.reset +
					"Usage: ./bandaid [args]\n" +
					"\nCommands:\n" +
					"-h | --help			Display help\n" +
					"-c | --no-ipchairs		Disable IpChairs\n" +
					"-n | --no-backup			Don't backup initial config\n" +
					"-r | --no-restore		Don't restore from backup\n" +
					"-f | --configfile [file]	Path for the config.json file\n" +
					"-b | --backup [folder]		Location of folder to store backups in\n" +
					// "-e | --no-encrypt		Don't encrypt backup folder\n" + //TODO add this
					"-q | --quiet			Disable output\n" +
					"-u | --upkeep			Disable upkeep of services\n" +
					"-p | --no-perms			Disable permission checking (faster)\n" +
					"-d | --delay [n]		Set interval to n\n" +
					"-i | --icmpdelay [n]		Set ICMP delay to n\n" +
					"\n",
			)
			os.Exit(0)
		case "-c", "--no-ipchairs":
			config.ipChairs = false
		case "-n", "--no-backup":
			config.doBackup = false
		case "-e", "--no-encrypt":
			config.doEncryption = false
		case "-r", "--no-restore":
			config.loadFromConfig = false
		case "-q", "--quiet":
			config.outputEnabled = false
		case "-p", "--no-perms":
			config.checkPerms = false
		case "-u", "--upkeep":
			config.upkeep = false
		case "-f", "--configfile":
			if i <= len(os.Args)-2 {
				config.configFile = os.Args[i+2]
			} else {
				Errorf("Error: must provide a config file location to use with --configfile\n")
				os.Exit(-1)
			}
		case "-b", "--backup":
			if i <= len(os.Args)-2 {
				config.backupLocation = os.Args[i+2]
			} else {
				Errorf("Error: must provide a config file location to use with --configfile\n")
				os.Exit(-1)
			}
		}
	}
}

// Main function to handle user input
func InputCommand() {
	caret()
	for {
		reader := bufio.NewReader(os.Stdin)
		rawCmd, _ := reader.ReadString('\n')
		cmd := trim(rawCmd)
		args := strings.Split(cmd, " ")
		switch args[0] {
		case "exit":
			os.Exit(0)
		case "help":
			fmt.Printf(
				"Commands:\n" +
					"list\n" +
					"checksums\n" +
					"addservice [name] [binary_path] [service_path] [config_path]\n" +
					"addfile [name] [file]\n" +
					"addfolder [name] [path]\n" +
					"free [name|file]\n" +
					"icmpInterval [milliseconds]\n" +
					"interval [milliseconds]\n" +
					"ipchairs\n" +
					"quiet\n" +
					"verbose\n" +
					"upkeep [on|off]\n" +
					"perms [on|off]\n" +
					"help\n" +
					"exit\n",
			)
		case "quiet":
			config.outputEnabled = false
		case "verbose":
			config.outputEnabled = true
		case "ipchairs":
			ipchairs.Enter()
		case "list":
			Warnf("---Services---\n")
			for _, service := range master.Services {
				fmt.Printf("(%s)\n", service.Name)
				for _, name := range serviceNames {
					fmt.Printf("%s: %s\n", name, service.getAttr(name).Path)
				}
				fmt.Println()
			}
			Warnf("\n---Directories---\n")
			for _, dir := range master.Directories {
				fmt.Printf("(%s)\n", dir.Name)
				for _, file := range dir.files {
					if file.isDir {
						fmt.Println(file.Path + "/*")
					} else {
						fmt.Println(file.Path)
					}
				}
				fmt.Println()
			}
			Warnf("\n---Files---\n")
			for _, file := range master.Files {
				fmt.Printf("%s: %s\n", file.Name, file.Path)
			}
		case "checksums":
			PrintChecksums()
		case "interval":
			if len(args) != 2 {
				Errorf("Error: Invalid number of arguments provided\n")
				break
			}
			if args[1] == "default" {
				config.delay = 1000
			}
			i, err := strconv.Atoi(args[1])
			if err != nil {
				Errorf("Error: Invalid argument\n")
			} else {
				config.delay = time.Duration(i)
				fmt.Printf("Interval set to %d.\n", i)
			}
		case "icmpInterval":
			if len(args) != 2 {
				Errorf("Error: Invalid number of arguments provided\n")
				break
			}
			if args[1] == "default" {
				config.delay = 10
			}
			i, err := strconv.Atoi(args[1])
			if err != nil {
				Errorf("Error: Invalid argument\n")
			} else {
				config.icmpDelay = time.Duration(i)
				fmt.Printf("ICMP Interval set to %d.\n", i)
			}
		case "addfile":
			if len(args) == 3 {
				if !FileExists(args[2]) && !BackupExists(args[2]) {
					Errorf("%s: file not found\n", args[2])
					break
				}
				if CheckName(args[1]) {
					Errorf("Error: %s already exists\n", args[1])
					break
				}
				// Create the service object and initialize it
				file := ServiceObject{
					Name: args[1],
					Path: args[2],
				}
				if !file.InitSO() {
					break
				}
				file.InitBackup()
				master.Files = append(master.Files, file)
				fmt.Printf("Added %s\n", args[1])
			} else {
				Errorf("Error: Wrong number of arguments provided\n")
			}
		case "addfolder":
			if len(args) == 3 {
				// Create the directory object and initialize it
				newDir := Directory{
					Name:        args[1],
					Path:        args[2],
					isRecursive: true,
				}
				if !newDir.InitDir() {
					Errorf("%s: folder not found\n", args[2])
					break
				}
				master.Directories = append(master.Directories, newDir)
				fmt.Println("Folder added")
			} else {
				Errorf("Error: Wrong number of arguments provided\n")
			}
		case "addservice":
			if len(args) == 5 {
				brk := false
				for _, arg := range args[2:] {
					if !FileExists(arg) {
						brk = true
						Errorf("%s: file not found\n", args[2])
						break
					}
				}
				if brk {
					break
				}
				if CheckName(args[1]) {
					Errorf("Error: %s already exists\n", args[1])
					break
				}
				// Create a service object for binary, service, and path,
				// then join them together with a Service
				binary := ServiceObject{
					Path: args[2],
				}
				service := ServiceObject{
					Path: args[3],
				}
				config := ServiceObject{
					Path: args[4],
				}
				serv := Service{
					Name:    args[1],
					Binary:  &binary,
					Service: &service,
					Config:  &config,
				}
				if !serv.Init() {
					Errorf("Error: Couldn't initialize service\n")
					break
				}
				for _, name := range serviceNames {
					serv.getAttr(name).InitBackup()
				}
				master.Services = append(master.Services, serv)
				fmt.Printf("Added %s\n", args[1])
			} else {
				Errorf("Error: Wrong number of arguments provided\n")
			}
		case "free": //TODO add filepath
			if len(args) > 1 {
				var removeList []int
				var fileRemoveList []int
				var dirRemoveList []int
				// Create a list for each type of object containing
				// which objects should be removed, then remove
				// them all at the end.
				for _, arg := range args[1:] {
					if CheckName(arg) {
						for e, service := range master.Services {
							if service.Name == arg {
								removeList = append(removeList, e)
								fmt.Printf("Removed %s\n", arg)
								break
							}
						}
						for e, file := range master.Files {
							if file.Name == arg || file.Path == arg {
								fileRemoveList = append(fileRemoveList, e)
								fmt.Printf("Removed %s\n", arg)
								break
							}
						}
						for e, dir := range master.Directories {
							if dir.Name == arg || dir.Path == arg {
								dirRemoveList = append(dirRemoveList, e)
								fmt.Printf("Removed %s\n", arg)
								break
							}
						}
					} else {
						Warnf("%s does not exist\n", arg)
					}
				}
				for _, i := range removeList {
					master.Services = removeService(master.Services, i)
				}
				for _, i := range fileRemoveList {
					master.Files = removeSO(master.Files, i)
				}
				for _, i := range dirRemoveList {
					master.Directories = removeDirectory(master.Directories, i)
				}
			} else {
				Errorf("Error: Not enough arguments\n")
			}
		case "upkeep":
			if len(args) != 2 {
				Errorf("Error: invalid number of arguments\n")
				break
			}
			switch args[1] {
			case "on":
				config.upkeep = true
			case "off":
				config.upkeep = false
			default:
				Errorf("Error: invalid argument")
			}
		case "perms":
			if len(args) != 2 {
				Errorf("Error: invalid number of arguments\n")
				break
			}
			switch args[1] {
			case "on":
				config.checkPerms = true
			case "off":
				config.checkPerms = false
			default:
				Errorf("Error: invalid argument")
			}
		case "":
		default:
			Errorf("Unknown command\n")
		}
		caret()
	}
}

// Print the names and checksums of each service, directory, and file
func PrintChecksums() {
	Warnf("---Services---\n")
	for _, service := range master.Services {
		fmt.Printf("(%s)\nConfig checksum: %s\nBinary checksum: %s\nService checksum: %s\n\n", service.Name, service.Config.Checksum, service.Binary.Checksum, service.Service.Checksum)
	}
	Warnf("---Files nested in directories---\n")
	for _, dir := range master.Directories {
		for _, file := range dir.files {
			if !file.isDir {
				fmt.Printf("%s: %s\n", file.Path, file.Checksum)
			}
		}
	}
	Warnf("\n---Files---\n")
	for _, file := range master.Files {
		fmt.Printf("%s: %s\n", file.Path, file.Checksum)
	}
}

// Main run function for bandaid
func RunBandaid() {
	for {
		// Keep a record of whether or not any changes were made
		change := false
		// Lock the mutex to make sure we don't read files while they're being freed
		isFreeing.Lock()
		for _, service := range master.Services {
			for _, name := range serviceNames {
				// First check each file's checksum. This also checks for file deletions
				if !service.getAttr(name).CheckFile() {
					if config.outputEnabled {
						fmt.Printf("\nError on checksum for %s %s. Rewriting...\n", service.Name, strings.ToLower(name))
						if service.getAttr(name).writeBackup() {
							fmt.Println("Backup succeeded.")
						} else {
							fmt.Println("Backup failed.")
						}
					} else {
						service.getAttr(name).writeBackup()
					}
					change = true
					// If the checksum was fine, also check the permissions (if enabled)
				} else if !service.getAttr(name).CheckPerms() && config.checkPerms {
					if service.getAttr(name).WritePerms() {
						fmt.Println("Permissions restored.")
					} else {
						fmt.Println("Error restoring permissions.")
					}
					change = true
				}
			}
			if config.upkeep {
				// Get the service name using the path
				serv := GetTail(service.Service.Path, "/")
				// Check to see if the service is running; if not, restart
				if !CheckCtl(serv) {
					fmt.Printf("\nService %s has stopped. Restarting...\n", service.Name)
					cmd := exec.Command("systemctl", "start", serv)
					cmd.Run()
					change = true
				}
			}
		}

		// Next, check all files
		for _, file := range master.Files {
			if !file.CheckFile() {
				if config.outputEnabled {
					fmt.Printf("\nError on checksum for %s (%s). Rewriting...\n", file.Name, file.Path)
					if file.writeBackup() {
						fmt.Println("Backup succeeded.")
					} else {
						fmt.Println("Backup failed.")
					}
				} else {
					file.writeBackup()
				}
				change = true
			} else if !file.CheckPerms() && config.checkPerms {
				if file.WritePerms() {
					fmt.Println("Permissions restored.")
				} else {
					fmt.Println("Error restoring permissions.")
				}
				change = true
			}
		}

		// Check every file in each directory (recursively)
		for _, dir := range master.Directories {
			// The Directory struct stores each file's path,
			// so no need to do recursive iteration
			for _, file := range dir.files {
				if !file.CheckFile() {
					if config.outputEnabled {
						fmt.Printf("\nError on checksum for %s. Rewriting...\n", file.Path)
						if file.writeBackup() {
							fmt.Println("Backup succeeded.")
						} else {
							fmt.Println("Backup failed.")
						}
					} else {
						file.writeBackup()
					}
					change = true
				} else if !file.CheckPerms() && config.checkPerms {
					if file.WritePerms() {
						fmt.Println("Permissions restored.")
					} else {
						fmt.Println("Error restoring permissions.")
					}
					change = true
				}
			}
		}
		// Unlock the mutex
		isFreeing.Unlock()
		// If there was a change made, then we need to caret()
		// because of the change output
		if change {
			caret()
		}
		time.Sleep(config.delay * time.Millisecond)
	}
}

// Function to check for the most common ICMP break
func FixICMP() {
	for {
		// This only works for linux
		// TODO: add a similar function for windows?
		if runtime.GOOS == "linux" {
			// If icmp_echo_ignore_all is not 0, set it back to 0
			if trim(readFile("/proc/sys/net/ipv4/icmp_echo_ignore_all")) != "0" {
				cmd := exec.Command("/bin/sh", "-c", "echo 0 > /proc/sys/net/ipv4/icmp_echo_ignore_all")
				cmd.Run()
				if config.outputEnabled {
					fmt.Println("\nICMP change detected; Re-enabled ICMP")
					caret()
				}
			}
		} else {
			return
		}
		time.Sleep(config.icmpDelay * time.Millisecond)
	}
}

// Initialize the backups for services and files.
// Directories are handled in the InitConfig function
func InitBackups() {
	for i := range master.Services {
		for _, name := range serviceNames {
			master.Services[i].getAttr(name).InitBackup()
		}
	}
	for i := range master.Files {
		master.Files[i].InitBackup()
	}
}

// Initialize the global config
func InitConfig() Services {
	configFile, err := os.Open(config.configFile)
	var configBytes []byte
	if err != nil {
		// If we can't find the config file, load the default config
		// from the config.go file, converting it into bytes
		// so we can unmarshal it
		Errorf("Could not load config file. Load default config? [y/n]: ")
		if GetInput() == "y" {
			configBytes = []byte(defaultConfig)
		} else {
			Errorf("No config found. Exiting...\n")
			os.Exit(-1)
		}
	} else {
		configBytes, _ = ioutil.ReadAll(configFile)
		defer configFile.Close()
	}
	var names []string
	var master Services
	json.Unmarshal(configBytes, &master)
	// Create a list of services, files and directories to not include
	var removeList []int
	var fileRemoveList []int
	var dirRemoveList []int
	// Initialize all services and add the ones that error to the removeList
	for i := range master.Services {
		if !master.Services[i].Init() {
			removeList = append(removeList, i)
		} else if contains(names, master.Services[i].Name) {
			fmt.Printf("Config error: Duplicate name (%s)\n", master.Services[i].Name)
			os.Exit(-1)
		} else {
			names = append(names, master.Services[i].Name)
		}
	}
	for i := range master.Files {
		if !master.Files[i].InitSO() {
			fileRemoveList = append(fileRemoveList, i)
		} else if contains(names, master.Files[i].Name) {
			Errorf("Config error: Duplicate name (%s)\n", master.Files[i].Name)
			os.Exit(-1)
		} else {
			names = append(names, master.Files[i].Name)
		}
	}
	for i := range master.Directories {
		if contains(names, master.Directories[i].Name) {
			Errorf("Config error: Duplicate name (%s)\n", master.Directories[i].Name)
			os.Exit(-1)
		} else {
			if master.Directories[i].InitDir() {
				names = append(names, master.Directories[i].Name)
			} else {
				dirRemoveList = append(dirRemoveList, i)
			}
		}
	}

	// Add all items that didn't error out to the global master struct
	var newServices []Service
	var newFiles []ServiceObject
	var newDirs []Directory
	for i := range master.Services {
		if !containsInt(removeList, i) {
			newServices = append(newServices, master.Services[i])
		}
	}
	for i := range master.Files {
		if !containsInt(fileRemoveList, i) {
			newFiles = append(newFiles, master.Files[i])
		}
	}
	for i := range master.Directories {
		if !containsInt(dirRemoveList, i) {
			newDirs = append(newDirs, master.Directories[i])
		}
	}
	master.Services = newServices
	master.Files = newFiles
	master.Directories = newDirs
	return master
}

// Create a nil file, to be used if a service doesn't have a config/binary/etc file.
// Located in C:\nil for windows and /dev/nil for linux.
func CreateNil() {
	if runtime.GOOS == "windows" {
		f, err := os.Create("C:\\nil")
		if err != nil {
			Errorf("Could not create C:\\nil. Are you running as administrator?\n")
			os.Exit(-1)
		}
		defer f.Close()
	} else if runtime.GOOS == "linux" {
		f, err := os.Create("/dev/nil")
		if err != nil {
			Errorf("Could not create /dev/nil. Are you root?\n")
			os.Exit(-1)
		}
		defer f.Close()
	}

}

// Check to see if a service/file/directory name already exists in the gloabl master
func CheckName(name string) bool {
	exists := false
	for _, service := range master.Services {
		if service.Name == name {
			exists = true
			break
		}
	}
	for _, file := range master.Files {
		if file.Name == name {
			exists = true
			break
		}
	}
	for _, dir := range master.Directories {
		if dir.Name == name {
			exists = true
			break
		}
	}
	return exists
}

// Check to see if a file's path already exists in the global master
func CheckPath(path string) bool {
	exists := false
	for _, service := range master.Services {
		for _, name := range serviceNames {
			if service.getAttr(name).Path == path {
				exists = true
				break
			}
		}

	}
	for _, file := range master.Files {
		if file.Path == path {
			exists = true
			break
		}
	}
	// TODO add directories as well
	return exists
}

// Check to see if a linux service is currently running or not
func CheckCtl(service string) bool {
	cmd := exec.Command("systemctl", "check", service)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if config.outputEnabled {
				if (exitErr.String()) == "3" {
					Warnf("\nSystemctl status 3 for service %s\n", service)
				} else {
					Warnf("\nSystemctl finished with non-zero for service %s: %v\n", service, exitErr)
				}
				// caret()
			}
		} else {
			if config.outputEnabled {
				Errorf("\nFailed to run systemctl: %v\n", err)
				config.upkeep = false
			}
		}
	}
	return trim(string(out)) == "active"
}

// CheckCtl but for IpChairs
func (a *IpChairs) CheckCtl(service string) bool {
	cmd := exec.Command("systemctl", "check", service)
	out, _ := cmd.CombinedOutput()
	return trim(string(out)) == "active"
}
