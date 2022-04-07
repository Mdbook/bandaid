package main

import "time"

type Config struct {
	delay          time.Duration
	icmpDelay      time.Duration
	configFile     string
	backupLocation string
	outputEnabled  bool
	loadFromConfig bool
	upkeep         bool
	doBackup       bool
	checkPerms     bool
}
