package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/getlantern/systray"
	"github.com/sc7639/mysql-notifier/icon"

	toast "gopkg.in/toast.v1"
	yaml "gopkg.in/yaml.v2"
)

func readSettings(readSettings chan settings, mysqlInstances chan mysql) (bool, error) { // Read settings
	data, err := ioutil.ReadFile(settingsPath)
	if err != nil {
		log.Printf("Failed to read settings: %s\n", err.Error())
		return false, err
	}

	s := settings{}
	err = yaml.Unmarshal(data, &s)
	if err != nil {
		log.Printf("Failed to read settings: %s\n", err.Error())
		return false, err
	}

	mysqlInstances <- s.Mysql
	readSettings <- s

	return true, nil
}

func openSettings(editor string) {
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) { // If settings file doesn't exist, create it and add min settings
		fp, err := os.OpenFile(settingsPath, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			log.Printf("Failed to open / create settings.yml: %s\n", err.Error())
			return
		}

		defer fp.Close()

		ys, err := yaml.Marshal(minSettings)
		if err != nil {
			log.Printf("Failed to create yml: %s\n", err.Error())
		}

		_, err = fp.Write(ys)
		if err != nil {
			log.Printf("Failed to write to settings.yml: %s\n", err.Error())
		}
	}

	cmd := exec.Command(editor, settingsPath)
	err := cmd.Start()
	if err != nil {
		log.Printf("Failed to open settings: %s\n", err.Error())
	} else {
		log.Println("Open Settings")
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
		port = details["port"]
	}

	var cmd *exec.Cmd
	if passwd := details["password"]; passwd != "" { // If password is set then execute command with password
		log.Println("Password")
		cmd = exec.Command("cmd", "/c", "start", "mysql", "-h"+host, "-u"+details["username"], "-p"+details["password"], "-P"+port)
	} else {
		log.Println("No Password")
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
		log.Printf("Failed to open mysql command line: %s", err)
	} else {
		log.Println("Open mysql cmd")
	}

	out, _ := ioutil.ReadAll(stdout)
	sterr, _ := ioutil.ReadAll(stderr)

	// log.Println("cmd /c start mysql -h"+host+" -u"+details["username"], " -P"+port)

	if string(out) != "" {
		log.Printf("Out pipe: %s\n", out)
	}
	if string(sterr) != "" {
		log.Printf("Err pipe: %s\n", sterr)
	}
}

func updateItem(status chan bool, title string, item *systray.MenuItem, appID string) { // On check of database connection update menu item title
	for live := range status {
		// log.Printf("Checked %s status: %v\n", title, live)
		if live {
			item.SetTitle(title + " - Live")
		} else {
			item.SetTitle(title + " - Dead")
		}
	}
}

// For each db status check, check check if the the status has changed and if so
func notifications(statuses []chan bool, appID string, connections mysql, duration time.Duration) {
	var prevStatuses = make([]bool, len(statuses))
	var changed = make([]bool, len(statuses))
	for i := range prevStatuses { // Initialize prevStatuses as true
		prevStatuses[i] = true
	}

	var notification toast.Notification
	var timer *time.Timer
	var connectionKeys []string
	var timerStarted bool
	sendNotification := make(chan []bool)

	for key := range connections {
		connectionKeys = append(connectionKeys, key)
	}

	for i, status := range statuses {
		go func(status chan bool, i int) {
			for live := range status {
				if !timerStarted {
					timer = time.NewTimer(duration)
					timerStarted = true
					log.Printf("duration: %v\n", duration)

					go func(sendNot chan<- []bool) {
						<-timer.C
						changedArr := make([]bool, len(changed))
						copy(changedArr, changed)
						sendNot <- changedArr

						timerStarted = false
						for i := range changed {
							changed[i] = false
						}
					}(sendNotification)
				}

				if prevStatuses[i] != live {
					changed[i] = true
					prevStatuses[i] = live
				}

			}
		}(status, i)
	}

	go func(sendNot <-chan []bool, appID string) {
		for {
			changedItems := <-sendNot
			changedIndices := make(map[string][]int)

			for i, changed := range changedItems {
				if changed {
					if prevStatuses[i] {
						changedIndices["live"] = append(changedIndices["live"], i)
					} else {
						changedIndices["dead"] = append(changedIndices["dead"], i)
					}
				}
			}

			// Create message and title depending on how many of the connections are down / up
			for typ := range changedIndices {
				title := ""
				message := "Failed to ping"
				numType := len(changedIndices[typ])
				icon := path + "\\images\\icon-red.png"
				// actions := []toast.Action{}

				if typ == "live" {
					message = "Successfully pinged "
					icon = path + "\\images\\icon.png"
				}

				if numType > 1 {
					title = strconv.Itoa(numType) + " Connections are " + typ
					title = strings.Title(title)
					message += "\n"
					for i := range changedIndices[typ] {
						message += strings.Title(connectionKeys[i]) + ", "
					}
					message += " databases"
				} else {
					title = connectionKeys[changedIndices[typ][0]] + " - " + typ
					title = strings.Title(title)
					message += title + " database"
					// actions = []toast.Action{
					// 	{Type: "protocol", Label: "SSH To Server", Arguments: ""},
					// }
				}

				if runtime.GOOS == "windows" {
					notification = toast.Notification{
						AppID:    appID,
						Icon:     icon,
						Title:    title,
						Message:  message,
						Duration: "long",
						// Actions: actions,
					}
				}

				err := notification.Push()
				if err != nil {
					log.Fatalln(err)
				}

			}
		}
	}(sendNotification, appID)

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

				if i == len(statuses) { // After receiving on all channels in the array
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
