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

var ipchairs IpChairs

type IpConfig struct {
	demo             bool
	safeMode         bool
	onlyFlush        bool
	disableFirewalls bool
	flushAllAllow    bool
	preDrop          bool
	preKill          bool
	basicFlush       bool
	allowEstablished bool
	allowICMP        bool
	enabled          bool
}

type IpChains struct {
	nat    []string
	mangle []string
	filter []string
	raw    []string
	none   []string
}

type IpChairs struct {
	tcp         []string
	udp         []string
	tables      []string
	flushtables []string
	locations   []IpChains
	chains      IpChains
	config      *IpConfig
}

func InitIpChairs() {
	ipchairs = IpChairs{}
	ipchairs.Init()
}

func (a *IpChairs) Init() {
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
		enabled:          true,
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
	a.chains = IpChains{
		nat:    []string{"PREROUTING", "POSTROUTING", "INPUT", "OUTPUT"},
		mangle: []string{"PREROUTING", "POSTROUTING", "INPUT", "OUTPUT", "FORWARD"},
		filter: []string{"INPUT", "OUTPUT", "FORWARD"},
		raw:    []string{"PREROUTING", "OUTPUT"},
		none:   []string{"INPUT", "OUTPUT", "FORWARD"},
	}
	a.tcp = []string{
		"22",
		"80",
	}
	a.udp = []string{}
}

func (a *IpChairs) Enter() {
	a.PrintHelp()
	a.caret()
	for {
		reader := bufio.NewReader(os.Stdin)
		rawCmd, _ := reader.ReadString('\n')
		cmd := trim(rawCmd)
		args := strings.Split(cmd, " ")
		switch args[0] {
		case "help":
			a.PrintHelp()
		case "exit":
			return
		case "d", "drop-first":
			if len(args) == 2 {
				switch args[2] {
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
				switch args[2] {
				case "on":
					a.config.onlyFlush = true
				case "off":
					a.config.onlyFlush = false
				default:
					Errorf("Syntax error\n")
				}
			} else {
				str := "off"
				if a.config.basicFlush {
					str = "on"
				}
				fmt.Printf("Flush only is %s\n", str)
			}
		case "i", "iron-wall":
			if len(args) == 2 {
				switch args[2] {
				case "on":
					a.config.safeMode = false
				case "off":
					a.config.safeMode = true
				default:
					Errorf("Syntax error\n")
				}
			} else {
				str := "off"
				if a.config.basicFlush {
					str = "on"
				}
				fmt.Printf("Iron wall is %s\n", str)
			}
		case "p", "ignore-ping":
			if len(args) == 2 {
				switch args[2] {
				case "on":
					a.config.allowICMP = false
				case "off":
					a.config.allowICMP = true
				default:
					Errorf("Syntax error\n")
				}
			} else {
				str := "off"
				if a.config.basicFlush {
					str = "on"
				}
				fmt.Printf("Ignore ping is %s\n", str)
			}
		case "enable":
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
		a.Run()
	}
}

func (a *IpChairs) Run() {
	if a.config.disableFirewalls {
		a.DisableFirwalls()
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
	tcp := strings.Join(a.tcp, ",")
	udp := strings.Join(a.udp, ",")
	for _, table := range a.tables {
		curChain := a.chains.Get(table)
		for _, chain := range curChain {
			a.Exec("-t " + table + " -A " + chain + " -m state --state ESTABLISHED,RELATED -j ACCEPT")
			if a.config.allowEstablished {
				a.Exec("-t " + table + " -A " + chain + " -m state --state ESTABLISHED,RELATED -j ACCEPT")
			}
			if tcp != "" {
				a.Exec("-t " + table + " -A " + chain + " -p tcp -m tcp -m multiport ! --dports " + tcp + " -j DROP")
			}
			if udp != "" {
				a.Exec("-t " + table + " -A " + chain + " -p udp -m udp -m multiport ! --dports " + udp + " -j DROP")
			}
			if !a.config.allowICMP {
				a.Exec("-t " + table + " -A " + chain + " -p icmp -j DROP")
			}
		}
	}
}

func (a *IpChairs) IronWall() {
	tcp := strings.Join(a.tcp, ",")
	udp := strings.Join(a.udp, ",")
	for _, table := range a.tables {
		curChain := a.chains.Get(table)
		for _, chain := range curChain {
			if tcp != "" {
				a.Exec("-t " + table + " -A " + chain + " -p tcp -m tcp -m multiport --dports " + tcp + " -j ACCEPT")
			}
			if udp != "" {
				a.Exec("-t " + table + " -A " + chain + " -p udp -m udp -m multiport --dports " + udp + " -j ACCEPT")
			}
			if a.config.allowEstablished {
				a.Exec("-t " + table + " -A " + chain + " -m state --state ESTABLISHED,RELATED -j ACCEPT")
			}
			if a.config.allowICMP {
				a.Exec("-t " + table + " -A " + chain + " -p icmp --icmp-type echo-request -j ACCEPT")
			}
			a.Exec("-t " + table + " -P " + chain + " DROP")
		}
	}
}
func (a *IpChairs) DisableFirwalls() {
	a.SysExec("ufw", "disable")
	if CheckCtl("firewalld") {
		a.SysExec("systemctl", "disable firewalld")
		a.SysExec("systemctl", "stop firewalld")
	}
}

func (a *IpChairs) PreDrop() {
	for _, table := range a.flushtables {
		curChain := a.chains.Get(table)
		for _, chain := range curChain {
			a.Exec("-t " + table + " -P " + chain + " DROP")
		}
	}
	time.Sleep(1 * time.Second)
}

func (a *IpChairs) FlushallAllow() {
	if !a.config.basicFlush {
		for _, table := range a.flushtables {
			a.Exec("-Z -t " + table)
			a.Exec("-F -t " + table)
			a.Exec("-X -t " + table)
		}
	} else {
		for _, table := range a.flushtables {
			a.Exec("-F -t " + table)
		}
	}
}

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
	case "none":
		return a.none
	}
	return nil
}

func (a *IpChairs) SysExec(binary string, command string) {
	args := strings.Split(command, " ")
	if a.config.demo {
		// fmt.Printf("%s %s\n", binary, command)
		return
	}
	cmd := exec.Command(binary, args...)
	cmd.Run()
}

func (a *IpChairs) Exec(command string) {
	args := strings.Split(command, " ")
	if a.config.demo {
		// fmt.Printf("iptables %s\n", command)
		return
	}
	cmd := exec.Command("iptables", args...)
	cmd.Run()
}
