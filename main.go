package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/getlantern/systray"
	"github.com/sc7639/mysql-notifier/icon"
	"github.com/sc7639/mysql-notifier/status"
)

var settingsFolder string
var settingsPath string

func init() {
	var appData string
	if runtime.GOOS == "windows" {
		appData = os.Getenv("APPDATA")
		settingsFolder = appData + "/mysql-notifier"

	} else {
		appData = os.Getenv("HOME")
		settingsFolder = appData + "/.mysql-notifier"
	}

	if _, err := os.Stat(settingsFolder); os.IsNotExist(err) {
		os.Mkdir(settingsFolder, 0755)
	}

	settingsPath = settingsFolder + "/settings.yml"
}

type settings struct {
	Mysql    map[string]map[string]string `yml:"mysql"`
	Interval string                       `yml:"interval"`
}

var minSettings = settings{
	Mysql:    map[string]map[string]string{},
	Interval: "10s",
}

func main() {
	systray.Run(onReady)
}

func onReady() { // Set icon title and add menu items
	systray.SetTitle("MySQL Notifier")

	rdSettings := make(chan settings, 1)
	mI := make(chan map[string]map[string]string, 1)

	go addMenuItems(mI, rdSettings) // Add menu items

	go func() { // Try to read settings and if settings file doesn't exist create one and wait 30 seconds before trying to read it again
		defer close(rdSettings)
		defer close(mI)

		loaded, err := readSettings(rdSettings, mI)
		if loaded == false || err != nil {
			systray.SetIcon(icon.Red)
			systray.SetTooltip("Failed to load settings")

			// Try to open settings ever 30 seconds
			for {
				loaded, err := readSettings(rdSettings, mI)
				if loaded == false || err != nil {
					go openSettings()
				} else {
					systray.SetIcon(icon.Green)
					systray.SetTooltip("All OK")
					break
				}

				time.Sleep(time.Second * 30) // Sleep for 30 seconds
			}
		} else {
			systray.SetIcon(icon.Green)
			systray.SetTooltip("All OK")
		}
	}()
	// fmt.Printf("rdSettings: %v", <-rdSettings)
}

func addMenuItems(mysqlInstance chan map[string]map[string]string, rdSettings chan settings) {
	ldSettings := <-rdSettings

	dbStatuses := make([]chan bool, len(ldSettings.Mysql))
	var i = 0
	for instance, details := range <-mysqlInstance { // For each mysql instance create a new menu item
		instance = strings.Title(instance)
		item := systray.AddMenuItem(instance, instance)

		go func(title string, item *systray.MenuItem, details map[string]string) { // Handle on click of mysql instance menu item
			for {
				<-item.ClickedCh
				openMysqlCMD(details)
			}
		}(instance, item, details)

		// Create database connection
		var host string
		if _, ok := details["host"]; !ok {
			host = "127.0.0.1"
		} else {
			host = details["host"]
		}

		var port string
		if _, ok := details["port"]; !ok {
			port = "3306"
		} else {
			port = details["port"]
		}

		dbConn, err := sql.Open("mysql", details["username"]+":"+details["password"]+"@"+"tcp("+host+":"+port+")/"+details["database"])
		if err != nil {
			log.Fatalf("Failed to open %s database connection: %s", instance, err.Error())
		}

		// Parse interval
		var interval time.Duration
		if ok := strings.Contains(ldSettings.Interval, "s"); ok {
			pInt, err := strconv.Atoi(strings.Replace(ldSettings.Interval, "s", "", -1))
			if err != nil {
				log.Panicf("Failed to parse interval format: %s", err.Error())
			}

			interval = time.Second * time.Duration(pInt)
		} else if ok := strings.Contains(ldSettings.Interval, "ms"); ok {
			pInt, err := strconv.Atoi(strings.Replace(ldSettings.Interval, "ms", "", -1))
			if err != nil {
				log.Panicf("Failed to parse interval format: %s", err.Error())
			}

			interval = time.Millisecond * time.Duration(pInt)
		} else if ok := strings.Contains(ldSettings.Interval, "m"); ok {
			pInt, err := strconv.Atoi(strings.Replace(ldSettings.Interval, "m", "", -1))
			if err != nil {
				log.Panicf("Failed to parse interval format: %s", err.Error())
			}

			interval = time.Minute * time.Duration(pInt)
		}

		dbStatus := make(chan bool)
		updateItemCH := make(chan bool)
		updateIconCH := make(chan bool)
		dbStatuses[i] = updateIconCH

		// Check database conenction
		go status.Check(dbConn, interval, dbStatus)

		go func() { // On db status channel update, push update to update item and update icon channels
			for live := range dbStatus {
				updateItemCH <- live
				updateIconCH <- live
			}
		}()

		go updateItem(updateItemCH, instance, item)

		i++
	}

	go updateIcon(dbStatuses)

	mOpenSettings := systray.AddMenuItem("Open Settings", "Settings")

	go func() { // Handle on click menu item handlers
		for {
			select {
			case <-mOpenSettings.ClickedCh: // On open settings click
				go openSettings()
			}
		}
	}()

	mExit := systray.AddMenuItem("Exit", "Exit Notifier")
	go func() { // On exit menu item click chanel read, exit system tray and app
		<-mExit.ClickedCh
		systray.Quit()
		fmt.Println("Exited")
		os.Exit(0)
	}()
}
