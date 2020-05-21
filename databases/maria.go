package maria

import (
	"database/sql"
	"sync"
)

var Database *sql.DB
var DBMutex = &sync.Mutex{}

func ExecSafe(query string, values ...interface{}) (sql.Result, error) {
	DBMutex.Lock()
	defer DBMutex.Unlock()
	result, err := Database.Exec(query, values...)
	if err != nil {
		return nil, err
	}
	return result, nil
}
