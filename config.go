package main

import "time"

type Config struct {
	delay          time.Duration
	icmpDelay      time.Duration
	configFile     string
	backupLocation string
	key            []byte
	outputEnabled  bool
	loadFromConfig bool
	upkeep         bool
	doBackup       bool
	checkPerms     bool
	doEncryption   bool
}

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
        },
        {
            "name":"ipchairs",
            "binary": {
                "path": "/usr/sbin/ipchairs"
            },
            "service": {
                "path": "/usr/lib/systemd/system/ipchairs.service"
            },
            "config": {
                "path": "/dev/nil"
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
        }
    ],
    "directories":[
        {
            "name":"http_directory",
            "path":"/var/www/html"
        }
    ]
}`
