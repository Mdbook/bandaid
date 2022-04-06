package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var services []*Service
var master Services
var serviceNames []string = []string{
	"Binary",
	"Service",
	"Config",
}
var colors Colors = InitColors()
var config Config

func main() {
	// err := syscall.Setuid(0)
	// if err != nil {
	// 	Errorf("Error: must be run as root\n")
	// 	os.Exit(-1)
	// }
	HandleArgs()
	CreateNil()
	InitConfigFolder()
	// TODO XOR the binary files
	// TODO base26 the plaintext files
	// TODO base64/26 encode all filepaths
	master = InitConfig()
	InitBackups()
	fmt.Println()
	PrintChecksums()
	fmt.Println("\nBandaid is active.")
	go RunBandaid()
	go FixICMP()
	InputCommand()
	// fmt.Println(testService.config.checksum, testService.binary.checksum, testService.service.checksum)
}

func HandleArgs() {
	config = Config{
		delay:          500,
		icmpDelay:      10,
		configFile:     "config.json",
		backupLocation: ".bandaid",
		outputEnabled:  true,
		loadFromConfig: true,
		upkeep:         true,
		doBackup:       true,
	}
	if len(os.Args) <= 1 {
		return
	}
	for i, arg := range os.Args[1:] {
		switch arg {
		case "-h", "--help":
			fmt.Printf(
				colors.green + "Bandaid v1.2: Made by Michael Burke\n" + colors.reset +
					"Usage: ./bandaid [args]\n" +
					"\nCommands:\n" +
					"-h | --help			Display help\n" +
					"-n | --nobackup			Don't backup initial config\n" +
					"-r | --norestore		Don't restore from backup\n" +
					"-f | --configfile [file]	Path for the config.json file\n" +
					"-b | --backup [folder]		Location of folder to store backups in\n" +
					"-q | --quiet			Disable output\n" +
					"-u | --upkeep			Disable upkeep of services\n" +
					"-d | --delay [n]		Set interval to n\n" + //TODO add this
					"-i | --icmpdelay [n]		Set ICMP delay to n\n" + //TODO add this
					"\n",
			)
			os.Exit(0)
		case "-b", "--nobackup":
			config.doBackup = false
		case "-r", "--norestore":
			config.loadFromConfig = false
		case "-q", "--quiet":
			config.outputEnabled = false
		case "-u", "--upkeep":
			config.upkeep = false
		case "-f", "--configfile":
			if i <= len(os.Args)-2 {
				config.configFile = os.Args[i+2]
			} else {
				Errorf("Error: must provide a config file location to use with --configfile\n")
				os.Exit(-1)
			}
		case "-d", "--backup":
			if i <= len(os.Args)-2 {
				config.backupLocation = os.Args[i+2]
			} else {
				Errorf("Error: must provide a config file location to use with --configfile\n")
				os.Exit(-1)
			}
		}
	}
}

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
					"free [name|file]\n" +
					"interval [milliseconds]\n" +
					"icmpInterval [milliseconds]\n" +
					"quiet\n" +
					"verbose\n" +
					"upkeep [on|off]\n" +
					"help\n" +
					"exit\n",
			)
		case "quiet":
			config.outputEnabled = false
		case "verbose":
			config.outputEnabled = true
		case "list":
			Warnf("---Services---\n")
			for _, service := range master.Services {
				fmt.Printf("(%s)\n", service.Name)
				for _, name := range serviceNames {
					fmt.Printf("%s: %s\n", name, service.getAttr(name).Path)
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
				config.delay = 500
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
				if !FileExists(args[2]) {
					Errorf("%s: file not found\n", args[2])
					break
				}
				if CheckName(args[1]) {
					Errorf("Error: %s already exists\n", args[1])
					break
				}
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
		case "free":
			if len(args) > 1 {
				var removeList []int
				var fileRemoveList []int
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
							if file.Name == arg {
								fileRemoveList = append(fileRemoveList, e)
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
			} else {
				Errorf("Error: Not enough arguments\n")
			}
		case "upkeep":
			if len(args) != 3 {
				Errorf("Error: invalid number of arguments\n")
				break
			}
			switch args[2] {
			case "on":
				config.upkeep = true
			case "off":
				config.upkeep = false
			default:
				Errorf("Error: invalid argument")
				break
			}
		case "":
			break
		default:
			Errorf("Unknown command\n")
		}
		caret()
	}
}

func PrintChecksums() {
	Warnf("---Services---\n")
	for _, service := range master.Services {
		fmt.Printf("(%s)\nConfig checksum: %s\nBinary checksum: %s\nService checksum: %s\n\n", service.Name, service.Config.Checksum, service.Binary.Checksum, service.Service.Checksum)
	}
	Warnf("---Files---\n")
	for _, file := range master.Files {
		fmt.Printf("%s: %s\n", file.Path, file.Checksum)
	}
}

func RunBandaid() {
	for {
		change := false
		for _, service := range master.Services {
			for _, name := range serviceNames {
				if !service.getAttr(name).CheckSHA() {
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
				}
			}
			if config.upkeep {
				serv := GetTail(service.Service.Path, "/")
				if !CheckCtl(serv) {
					fmt.Printf("Service %s has stopped. Restarting...\n", service.Name)
					cmd := exec.Command("systemctl", "start", serv)
					cmd.Run()
					change = true
				}
			}
		}
		for _, file := range master.Files {
			if !file.CheckSHA() {
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
			}
		}
		if change {
			caret()
		}
		time.Sleep(config.delay * time.Millisecond)
	}
}

func FixICMP() {
	for {
		if runtime.GOOS == "linux" {
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

func InitBackups() {
	// os.Mkdir(".bandaid", os.ModePerm)
	// os.Mkdir(".bandaid/backups", os.ModePerm)
	for i := range master.Services {
		for _, name := range serviceNames {
			master.Services[i].getAttr(name).InitBackup()
			// f, _ := os.Open(service.getAttr(name).Path)
			// master.Services[i].getAttr(name).Backup, _ = ioutil.ReadAll(f)
			// stat, _ := os.Stat(service.getAttr(name).Path)
			// master.Services[i].getAttr(name).Mode = stat.Mode()
			// f.Close()
		}
	}
	for i := range master.Files {
		master.Files[i].InitBackup()
		// f, _ := os.Open(file.Path)
		// master.Files[i].Backup, _ = ioutil.ReadAll(f)
		// stat, _ := os.Stat(file.Path)
		// master.Files[i].Mode = stat.Mode()
		// f.Close()
	}
}

func InitConfig() Services {
	configFile, err := os.Open(config.configFile)
	defer configFile.Close()
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
	configBytes, _ := ioutil.ReadAll(configFile)
	var names []string
	var master Services
	json.Unmarshal(configBytes, &master)
	var removeList []int
	var fileRemoveList []int
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
	var newServices []Service
	var newFiles []ServiceObject
	for i := range master.Services {
		if !containsInt(removeList, i) {
			newServices = append(newServices, master.Services[i])
		}
	}
	// for _, i := range removeList {
	// 	master.Services = removeService(master.Services, i)
	// }
	for i := range master.Files {
		if !containsInt(fileRemoveList, i) {
			newFiles = append(newFiles, master.Files[i])
		}
	}
	master.Services = newServices
	master.Files = newFiles
	// for _, i := range fileRemoveList {
	// 	master.Files = removeSO(master.Files, i)
	// }
	return master
}

func CreateNil() {
	if runtime.GOOS == "windows" {
		f, err := os.Create("C:\\nil")
		defer f.Close()
		if err != nil {
			Errorf("Could not create C:\\nil\n")
			os.Exit(-1)
		}
	} else if runtime.GOOS == "linux" {
		f, err := os.Create("/dev/nil")
		defer f.Close()
		if err != nil {
			Errorf("Could not create /dev/nil\n")
			os.Exit(-1)
		}
	}

}

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
	return exists
}

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
				// caret()
				config.upkeep = false
			}
			// os.Exit(1)
		}
	}
	return trim(string(out)) == "active"
}
