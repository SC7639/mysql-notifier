package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/browser"

	_ "github.com/go-sql-driver/mysql"

	"github.com/getlantern/systray"
	"github.com/sc7639/mysql-notifier/icon"
	"github.com/sc7639/mysql-notifier/status"
)

var settingsFolder string
var settingsPath string
var path = filepath.Dir(os.Args[0])

func init() {
	var appData string
	if runtime.GOOS == "windows" {
		appData = os.Getenv("APPDATA")
		settingsFolder = appData + "\\mysql-notifier\\"

	} else {
		appData = os.Getenv("HOME")
		settingsFolder = appData + "/.mysql-notifier/"
	}

	if _, err := os.Stat(settingsFolder); os.IsNotExist(err) {
		os.Mkdir(settingsFolder, 0755)
	}

	settingsPath = settingsFolder + "settings.yml"
}

type mysql map[string]map[string]string
type settings struct {
	Mysql    mysql  `yml:"mysql"`
	Interval string `yml:"interval"`
	Editor   string `yml:"editor"`
	AppID    string `yml:"appid"`
}

var minSettings = settings{
	Mysql:    mysql{},
	Interval: "10s",
	Editor:   "",
}

func main() {
	systray.Run(onReady)
}

func onReady() { // Set icon title and add menu items
	systray.SetTitle("MySQL Notifier")

	rdSettings := make(chan settings, 1)
	mI := make(chan mysql, 1)

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
					systray.SetIcon(icon.Red)
					systray.SetTooltip("Failed to load settings")
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

func addMenuItems(mysqlInstances chan mysql, rdSettings chan settings) {
	ldSettings := <-rdSettings

	interval, err := time.ParseDuration(ldSettings.Interval)
	if err != nil {
		log.Fatal(err)
	}
	dbStatuses := make([]chan bool, len(ldSettings.Mysql))
	var i = 0
	for instance, details := range <-mysqlInstances { // For each mysql instance create a new menu item
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

		dbStatus := make(chan bool)
		updateItemCH := make(chan bool)
		updateIconCH := make(chan bool)
		dbStatuses[i] = updateIconCH

		// Check database connection
		go status.Check(dbConn, interval, dbStatus)

		go func() { // On db status channel update, push update to update item and update icon channels

			for live := range dbStatus {
				updateItemCH <- live
				updateIconCH <- live
			}
		}()

		go updateItem(updateItemCH, instance, item, ldSettings.AppID)

		i++
	}

	go notifications(dbStatuses, ldSettings.AppID, ldSettings.Mysql, interval)

	go updateIcon(dbStatuses)

	mOpenSettings := systray.AddMenuItem("Open Settings", "Settings")
	mEnableNotifications := systray.AddMenuItem("How To Enable Notifications", "Notifications")

	go func() { // Handle on click menu item handlers
		for {
			select {
			case <-mOpenSettings.ClickedCh: // On open settings click
				go openSettings(ldSettings.Editor)
			case <-mEnableNotifications.ClickedCh:
				browser.OpenURL("https://github.com/SC7639/mysql-notifier")
			}
		}
	}()

	mExit := systray.AddMenuItem("Exit", "Exit Notifier")
	go func() { // On exit menu item click channel read, exit system tray and app
		<-mExit.ClickedCh
		systray.Quit()
		fmt.Println("Exited")
		os.Exit(0)
	}()
}
