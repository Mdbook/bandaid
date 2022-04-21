package main

import "time"

// Define the config struct
type Config struct {
	delay           time.Duration // Delay interval for main checker
	icmpDelay       time.Duration // Delay interval for ICMP checker
	configFile      string        // Location of the config file
	backupLocation  string        // Folder to store the backups in. Default is .bandaid
	key             []byte        // AES key to be used for encryption
	outputEnabled   bool
	loadFromConfig  bool
	upkeep          bool // Toggle for maintaining of services
	doBackup        bool
	checkPerms      bool // Toggle for checking permissions and attributes of files
	doEncryption    bool
	ipChairs        bool
	ipChairsConsole bool // Toggle ipChairsConsole. Used in utils caret() function
}

// Default config. This can be exported into a .json file and modified as needed.
var defaultConfig string = `{
    "services": [
        {
            "name":"sshd_centos",
            "binary": {
                "path": "/usr/sbin/sshd"
            },
            "service": {
                "path": "/usr/lib/systemd/system/sshd.service"
            },
            "config": {
                "path": "/etc/ssh/sshd_config"
            }
        },
		{
            "name":"sshd_backup",
            "binary": {
                "path": "/usr/sbin/sshd"
            },
            "service": {
                "path": "/usr/lib/systemd/system/ssh.service"
            },
            "config": {
                "path": "/etc/ssh/sshd_config"
            }
        },
        {
            "name":"ftp_ubuntu",
            "binary": {
                "path": "/usr/sbin/vsftpd"
            },
            "service": {
                "path": "/usr/lib/systemd/system/vsftpd.service"
            },
            "config": {
                "path": "/etc/vsftpd.conf"
            }
        },
        {
            "name":"ftp_centos",
            "binary": {
                "path": "/usr/sbin/vsftpd"
            },
            "service": {
                "path": "/usr/lib/systemd/system/vsftpd.service"
            },
            "config": {
                "path": "/etc/vsftpd/vsftpd.conf"
            }
        },
        {
            "name":"http_centos",
            "binary": {
                "path": "/usr/sbin/httpd"
            },
            "service": {
                "path": "/usr/lib/systemd/system/httpd.service"
            },
            "config": {
                "path": "/etc/httpd/conf/httpd.conf"
            }
        },
        {
            "name":"http_ubuntu",
            "binary": {
                "path": "/usr/sbin/apache2"
            },
            "service": {
                "path": "/usr/lib/systemd/system/apache2.service"
            },
            "config": {
                "path": "/etc/apache2/apache2.conf"
            }
        }
    ],
    "other_files":[
        {
            "name":"Bash",
            "path":"/bin/bash"
        },
        {
            "name":"sh",
            "path":"/bin/sh"
        },
        {
            "name":"zsh",
            "path":"/bin/zsh"
        },
        {
            "name":"passwd",
            "path":"/etc/passwd"
        },
        {
            "name":"group",
            "path":"/etc/group"
        },
        {
            "name":"sudoers",
            "path":"/etc/sudoers"
        },
        {
            "name":"shadow",
            "path":"/etc/shadow"
        }
    ],
    "directories":[
        {
            "name":"http_directory",
            "path":"/var/www/html"
        }
    ]
}`
