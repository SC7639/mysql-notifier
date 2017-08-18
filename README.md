# mysql-notifier
Notification area (task bar) icon to display the status of one or more MySQL databases (live, dead)

## Installation
As a command line app
```
go get github.com/SC7639/mysql-notifier
```

As a gui less app (no cmd / terminal window for windows. Will add linux and mac commands in the future)
```
git clone https://github.com/SC7639/mysql-notifier.git
cd mysql-notifier
go build -ldflags -H=windowsgui -o mysql-notifier.exe main.go functionality.go
```
## Example
If all of the mysql database are live the icon (outlined in red)

![All Live](/images/readme-all-live.png)

If one of the mysql databases goes down (outlined in red)

![Dead](/images/readme-dead.png)

## Settings
Settings are stored in a yml file
```
mysql:
    dev:
        user: root
        database: mysql
     live:
         host: 192.168.0.0 #example
         port: 3306
         user: test
         password: tezt
         database: app_db
interval: 1m
```

#### MySQL options

- host (optional)
- port (optional)
- user
- password (optional)

#### Interval options

- 1m (number of minutes)
- 1s (number of seconds)
- 1ms (number of milliseconds)

## Funtionality

Checks the status of configured mysql databases and displays a green icon if live (can connect) or a red icon if dead (cant connect)

#### Menu

![Icon Menu](/images/readme-menu.png)

The menu adds extra functionality:

- Open settings (currently with notepad++, will add a text editor for linux and mac)
- Status of MySQL instance (live or dead)
- On MySQL menu item click, a mysql window will open with the connection details (currently only cmd on windows, will update )
