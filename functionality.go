package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/getlantern/systray"
	"github.com/sc7639/mysql-notifier/icon"

	yaml "gopkg.in/yaml.v2"
)

func readSettings(readSettings chan settings, mysqlInstances chan map[string]map[string]string) (bool, error) { // Read settings
	data, err := ioutil.ReadFile(settingsPath)
	if err != nil {
		fmt.Printf("Failed to read settings: %s\n", err.Error())
		return false, err
	}

	s := settings{}
	err = yaml.Unmarshal(data, &s)
	if err != nil {
		fmt.Printf("Failed to read settings: %s\n", err.Error())
		return false, err
	}

	mysqlInstances <- s.Mysql
	readSettings <- s

	return true, nil
}

func openSettings() {
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) { // If settings file doesn't exist, create it and add min settings
		fp, err := os.OpenFile(settingsPath, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			fmt.Printf("Failed to open / create settings.yml: %s\n", err.Error())
			return
		}

		defer fp.Close()

		ys, err := yaml.Marshal(minSettings)
		if err != nil {
			fmt.Printf("Failed to create yml: %s\n", err.Error())
		}

		_, err = fp.Write(ys)
		if err != nil {
			fmt.Printf("Failed to write to settings.yml: %s\n", err.Error())
		}
	}

	cmd := exec.Command("notepad++.exe", settingsPath)
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Failed to open settings: %s\n", err.Error())
	} else {
		fmt.Println("Open Settings")
	}
}

func openMysqlCMD(details map[string]string) {
	// Parse ip address
	var host string
	if _, ok := details["host"]; !ok {
		host = "127.0.0.1"
	} else {
		host = details["host"]
	}

	// Parse port
	var port string
	if _, ok := details["port"]; !ok {
		port = "3306"
	} else {
		port = "3316"
	}

	var cmd *exec.Cmd
	if passwd := details["password"]; passwd != "" {
		fmt.Println("Password")
		cmd = exec.Command("cmd", "/c", "start", "mysql", "-h"+host, "-u"+details["username"], "-p"+details["password"], "-P"+port)
	} else {
		fmt.Println("No Password")
		cmd = exec.Command("cmd", "/c", "start", "mysql", "-h"+host, "-u"+details["username"], "-P"+port)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		fmt.Printf("Failed to open mysql command line: %s", err)
	} else {
		fmt.Println("Open mysql cmd")
	}

	out, _ := ioutil.ReadAll(stdout)
	sterr, _ := ioutil.ReadAll(stderr)

	// log.Println("cmd /c start mysql -h"+host+" -u"+details["username"], " -P"+port)

	if string(out) != "" {
		fmt.Printf("Out pipe: %s\n", out)
	}
	if string(sterr) != "" {
		fmt.Printf("Err pipe: %s\n", sterr)
	}
}

func updateItem(status chan bool, title string, item *systray.MenuItem) { // On check of database connection update menu item title
	for live := range status {
		// fmt.Printf("Checked %s status: %v\n", title, live)
		if live {
			item.SetTitle(title + " - Live")
		} else {
			item.SetTitle(title + " - Dead")
		}
	}
}

func updateIcon(statuses []chan bool) { // On check of database connection update menu icon
	var allLive = true
	var i = 0
	for _, status := range statuses { // For each status
		go func(status chan bool) {
			for live := range status { // Wait to receive information on channel
				i++
				// log.Printf("status: %v, i: %v, len(statuses): %v", live, i, len(statuses))
				if !live {
					allLive = false
				}

				if i == len(statuses) { // After receivinv on all channels in the array
					if allLive {
						systray.SetIcon(icon.Green)
						systray.SetTooltip("All Ok")
					} else {
						systray.SetIcon(icon.Red)
						systray.SetTooltip("Connection Down")

						allLive = true
					}

					i = 0
				}
			}
		}(status)
	}
}
