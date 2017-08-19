package status

import (
	"database/sql"
	"time"
)

// Check function checks the status of the database every x interval
func Check(dbConn *sql.DB, interval time.Duration, status chan bool) {
	ticker := time.NewTicker(interval)

	dbConn.SetConnMaxLifetime(interval)

	// Check db immidatley
	pingDB(dbConn, status)

	// When interval has been reached check db connection
	for range ticker.C {
		pingDB(dbConn, status)
	}
}

func pingDB(dbConn *sql.DB, status chan bool) {
	err := dbConn.Ping()
	if err != nil {
		status <- false
	} else {
		status <- true
	}
}
