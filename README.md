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

![All Live](https://github.com/SC7639/mysql-notifier/blob/master/images/readme-all-live.png)

If one of the mysql databases goes down (outlined in red)

![Dead](https://github.com/SC7639/mysql-notifier/blob/master/images/readme-dead.png)

## Settings
Settings are stored in a yml file 
```
mysql:
    dev:
        user: root
        database: mysql
     live:
         ip: 192.168.0.0 #example
         port: 3306
         user: test
         password: tezt
         database: app_db
interval: 1m
```

#### MySQL options

- ip (optional)
- port (optional)
- user
- password (optional)
- database

#### Interval options

- 1m (number of minutes)
- 1s (number of seconds)
- 1ms (number of milliseconds)
