package status

import (
	"database/sql"
	"time"
)

// Check function checks the status of the database every x interval
func Check(dbConn *sql.DB, interval time.Duration, status chan bool) {
	ticker := time.NewTicker(interval)

	dbConn.SetConnMaxLifetime(interval)

	// When interval has been reached read time from ticker channel
	for _ = range ticker.C {
		// log.Println("check dbcon")
		err := dbConn.Ping()
		if err != nil {
			status <- false
		} else {
			status <- true
		}
	}
}
