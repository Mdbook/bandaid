/*
IpChairs- Made by Mikayla Burke
Module for bandaid to constantly set iptables rules to allow
certain ports and remove all other rules and chains.
*/

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Global ipchairs object
var ipchairs IpChairs

type IpChairs struct {
	tcp         []string  // Array of tcp ports to allow
	udp         []string  // Array of udp ports to allow
	tables      []string  // List of tables (filter, mangle, nat)
	flushtables []string  // List of tables to be used in flushing (filter, mangle, nat, raw)
	chains      IpChains  // IpChains object
	config      *IpConfig // IpConfig object
	// locations   []IpChains
}

type IpConfig struct {
	demo             bool // Setting this to true disables actually changing the rules. Used for debugging.
	safeMode         bool // Toggle IronWall on or off. More details on IronWall below
	onlyFlush        bool // Completely flush rules every iteration and don't create new ones
	disableFirewalls bool // Disable firewalld and ufw every iteration
	flushAllAllow    bool // Whether or not ipchairs should flush all existing rules before establishing rules
	preDrop          bool // Set all chains default policy to DROP first, then reset if safemode is on.
	preKill          bool // Attempt to drop all connections before establishing rules. Doesn't work very well. Don't use this.
	basicFlush       bool // Only use iptables -F to flush
	allowEstablished bool // Allow established/related connections
	allowICMP        bool // Allow ICMP in and out
	enabled          bool // Enable/disable ipchairs
}

type IpChains struct {
	nat    []string // List of chains in the nat table
	mangle []string // List of chains in the mangle table
	filter []string // List of chains in the filter table
	raw    []string // List of chains in the raw table
}

// Initialize the global variable and run the init script
func InitIpChairs() {
	ipchairs = IpChairs{}
	ipchairs.Init()
}

func (a *IpChairs) Init() {
	//Initialize default config
	a.config = &IpConfig{
		demo:             false,
		safeMode:         true,
		onlyFlush:        false,
		disableFirewalls: true,
		flushAllAllow:    true,
		preDrop:          false,
		preKill:          false,
		basicFlush:       false,
		allowEstablished: true,
		allowICMP:        true,
		enabled:          false,
	}
	a.tables = []string{
		"nat",
		"mangle",
		"filter",
	}
	a.flushtables = []string{
		"nat",
		"mangle",
		"filter",
		"raw",
	}
	a.tcp = []string{
		"22",
		"80",
	}
	//No UDP allowed by default
	a.udp = []string{}
	// Define valid chains for each table
	a.chains = IpChains{
		nat:    []string{"PREROUTING", "POSTROUTING", "INPUT", "OUTPUT"},
		mangle: []string{"PREROUTING", "POSTROUTING", "INPUT", "OUTPUT", "FORWARD"},
		filter: []string{"INPUT", "OUTPUT", "FORWARD"},
		raw:    []string{"PREROUTING", "OUTPUT"},
	}
}

// Enter the ipchairs menu
func (a *IpChairs) Enter() {
	a.PrintHelp()
	a.caret()
	for {
		//Read command
		reader := bufio.NewReader(os.Stdin)
		rawCmd, _ := reader.ReadString('\n')
		//Trim any newline characters. Not super necessary but better safe than sorry
		cmd := trim(rawCmd)
		args := strings.Split(cmd, " ")
		// Handle commands
		switch args[0] {
		case "help":
			a.PrintHelp()
		case "exit":
			return
		case "d", "drop-first":
			if len(args) == 2 {
				switch args[1] {
				case "on":
					a.config.basicFlush = true
				case "off":
					a.config.basicFlush = false
				default:
					Errorf("Syntax error\n")
				}
			} else {
				str := "off"
				if a.config.basicFlush {
					str = "on"
				}
				fmt.Printf("Drop first is %s\n", str)
			}
		case "f", "flush-only":
			if len(args) == 2 {
				switch args[1] {
				case "on":
					a.config.onlyFlush = true
				case "off":
					a.config.onlyFlush = false
				default:
					Errorf("Syntax error\n")
				}
			} else {
				str := "off"
				if a.config.onlyFlush {
					str = "on"
				}
				fmt.Printf("Flush only is %s\n", str)
			}
		case "i", "iron-wall":
			if len(args) == 2 {
				switch args[1] {
				case "on":
					a.config.safeMode = false
				case "off":
					a.config.safeMode = true
				default:
					Errorf("Syntax error\n")
				}
			} else {
				str := "off"
				if a.config.safeMode {
					str = "on"
				}
				fmt.Printf("Iron wall is %s\n", str)
			}
		case "p", "ignore-ping":
			if len(args) == 2 {
				switch args[1] {
				case "on":
					a.config.allowICMP = false
				case "off":
					a.config.allowICMP = true
				default:
					Errorf("Syntax error\n")
				}
			} else {
				str := "off"
				if a.config.allowICMP {
					str = "on"
				}
				fmt.Printf("Ignore ping is %s\n", str)
			}
		case "enable":
			// TODO make the enable/disable start and stop the goroutine
			a.config.enabled = true
		case "disable":
			a.config.enabled = false
		case "status":
			str := "disabled"
			if a.config.enabled {
				str = "enabled"
			}
			fmt.Printf("IpChairs is currently:  %s\n", str)
		case "l", "list":
			fmt.Printf(
				colors.yellow+"---Current Config---\n"+colors.reset+
					"Basic flush: %t\n"+
					"Drop first: %t\n"+
					"Flush only: %t\n"+
					"Iron wall: %t\n"+
					"Ignore Ping: %t\n"+
					"TCP Ports: %s\n"+
					"UDP Ports: %s\n"+
					"\n",
				a.config.basicFlush,
				a.config.preDrop,
				a.config.onlyFlush,
				!a.config.safeMode,
				!a.config.allowICMP,
				strings.Join(a.tcp, ","),
				strings.Join(a.udp, ","),
			)
		case "tcp", "udp":
			if len(args) < 2 {
				Errorf("Error: not enough arguments\n")
				break
			}
			var newPorts []string
			fail := false
			for _, port := range args[1:] {
				_, err := strconv.Atoi(port)
				if err != nil {
					fail = true
					break
				}
				newPorts = append(newPorts, port)
			}
			if !fail {
				if args[0] == "tcp" {
					a.tcp = newPorts
				} else {
					a.udp = newPorts
				}
			} else {
				Errorf("Syntax error\n")
			}
		case "":
		default:
			Errorf("Unknown Command")
		}

		a.caret()
	}
}

func (a *IpChairs) PrintHelp() {
	fmt.Printf( //TODO fix this
		colors.yellow + "---IpChairs v1.1- Made by Mikayla Burke---\n" + colors.reset +
			"b or basic-flush [on/off]   |   Only basic flush (iptables -F)\n" +
			"d or drop-first [on/off]    |   Drop all incoming connections for 1 second\n" +
			"                            |   before establishing new ones\n" +
			"f or flush-only [on/off]    |   Flush all rules; don't establish new ones\n" +
			"i or iron-wall [on/off]     |   Iron wall mode (Do NOT use with cloud boxes)\n" +
			"p or ignore-ping [on/off]   |   Block ICMP Ping requests\n" +
			"l or list                   |   List current settings\n" +
			"enable                      |   Enable IpChairs\n" +
			"disable                     |   Disable IpChairs \n" +
			"status                      |   Show current status\n" +
			"tcp [port1] [port2] ...     |   Specify which TCP ports to allow\n" +
			"udp [port1] [port2] ...     |   Specify which UDP ports to allow\n" +
			"exit                        |   Leave the IpChairs config terminal\n" +
			"\n",
	)
}

func (a *IpChairs) Start() {
	for {
		if a.config.enabled {
			a.Run()
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (a *IpChairs) Run() { // Run ipchairs
	if a.config.disableFirewalls {
		a.DisableFirewalls()
	}
	if a.config.flushAllAllow {
		a.FlushallAllow()
	}
	if a.config.preDrop {
		a.PreDrop()
	}
	if !a.config.onlyFlush {
		if a.config.safeMode {
			a.SafeMode()
		} else {
			a.IronWall()
		}
	}
}

func (a *IpChairs) SafeMode() {
	// Parse TCP and UDP strings so they can be put directly into the command
	tcp := strings.Join(a.tcp, ",")
	udp := strings.Join(a.udp, ",")
	// Cycle through all tables and chains
	for _, table := range a.tables {
		curChain := a.chains.Get(table)
		for _, chain := range curChain {
			// a.Exec("-t " + table + " -A " + chain + " -m state --state ESTABLISHED,RELATED -j ACCEPT")
			if a.config.allowEstablished {
				// Allow established connections first
				a.Exec("-t " + table + " -A " + chain + " -m state --state ESTABLISHED,RELATED -j ACCEPT")
			}
			if tcp != "" {
				//Only allow specified tcp ports
				a.Exec("-t " + table + " -A " + chain + " -p tcp -m tcp -m multiport ! --dports " + tcp + " -j DROP")
			}
			if udp != "" {
				// Only allow specified udp ports
				a.Exec("-t " + table + " -A " + chain + " -p udp -m udp -m multiport ! --dports " + udp + " -j DROP")
			}
			if !a.config.allowICMP {
				// Disallow ICMP connections if the option is enabled
				a.Exec("-t " + table + " -A " + chain + " -p icmp -j DROP")
			}
		}
	}
}

/*
Iron wall mode sets the default policy to DROP for all tables and chains,
and then only allows certain connections. This is technically a little more
secure but is NOT recommended for cloud boxes.
*/
func (a *IpChairs) IronWall() {
	// Parse TCP and UDP strings so they can be put directly into the command
	tcp := strings.Join(a.tcp, ",")
	udp := strings.Join(a.udp, ",")
	// Cycle through all tables and chains
	for _, table := range a.tables {
		curChain := a.chains.Get(table)
		for _, chain := range curChain {
			if tcp != "" {
				// Only allow specified tcp ports
				a.Exec("-t " + table + " -A " + chain + " -p tcp -m tcp -m multiport --dports " + tcp + " -j ACCEPT")
			}
			if udp != "" {
				// Only allow specified udp ports
				a.Exec("-t " + table + " -A " + chain + " -p udp -m udp -m multiport --dports " + udp + " -j ACCEPT")
			}
			if a.config.allowEstablished {
				// Allow established connections first
				a.Exec("-t " + table + " -A " + chain + " -m state --state ESTABLISHED,RELATED -j ACCEPT")
			}
			if a.config.allowICMP {
				// Allow ICMP connections if the option is enabled
				a.Exec("-t " + table + " -A " + chain + " -p icmp --icmp-type echo-request -j ACCEPT")
			}
			// Set the default policy to drop
			a.Exec("-t " + table + " -P " + chain + " DROP")
		}
	}
}

// Disable ufw and firewalld
func (a *IpChairs) DisableFirewalls() {
	a.SysExec("ufw", "disable")
	if a.CheckCtl("firewalld") {
		a.SysExec("systemctl", "disable firewalld")
		a.SysExec("systemctl", "stop firewalld")
	}
}

// Set firewall policy to drop first, if enabled
func (a *IpChairs) PreDrop() {
	for _, table := range a.flushtables {
		curChain := a.chains.Get(table)
		for _, chain := range curChain {
			a.Exec("-t " + table + " -P " + chain + " DROP")
		}
	}
	time.Sleep(1 * time.Second)
}

// Flush all rules before establishing
func (a *IpChairs) FlushallAllow() {
	if !a.config.basicFlush {
		// Iterate through all tables
		for _, table := range a.flushtables {
			// Zero the packet and byte counters in all chains
			a.Exec("-Z -t " + table)
			// Flush rules in all chains
			a.Exec("-F -t " + table)
			// Delete all chains except default
			a.Exec("-X -t " + table)
		}
	} else {
		// Iterate through all tables
		for _, table := range a.flushtables {
			// If basic flush is enabled, only flush rules
			a.Exec("-F -t " + table)
		}
	}
}

// Helper function since you can't call IpChains["foo"] in golang
func (a *IpChains) Get(field string) []string {
	switch field {
	case "nat":
		return a.nat
	case "mangle":
		return a.mangle
	case "filter":
		return a.filter
	case "raw":
		return a.raw
	}
	return nil
}

// Execute a system command
func (a *IpChairs) SysExec(binary string, command string) {
	args := strings.Split(command, " ")
	if a.config.demo {
		// fmt.Printf("%s %s\n", binary, command)
		return
	}
	cmd := exec.Command(binary, args...)
	cmd.Run()
}

// Execute an iptables command
func (a *IpChairs) Exec(command string) {
	args := strings.Split(command, " ")
	if a.config.demo {
		// fmt.Printf("iptables %s\n", command)
		return
	}
	cmd := exec.Command("iptables", args...)
	cmd.Run()
}
