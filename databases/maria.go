package maria

import (
	"database/sql"
	"sync"
)

var Database *sql.DB
var DBMutex = &sync.Mutex{}

func SafeExec(query string, values ...interface{}) (sql.Result, error) {
	result, err := Database.Exec(query, values...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func SafeQuery(query string, values ...interface{}) ([]string, error) {
	var resultRows []string
	rows, err := Database.Query(query, values...)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var tmpString string
		err := rows.Scan(&tmpString)
		if err != nil {
			continue
		}
		resultRows = append(resultRows, tmpString)
	}
	return resultRows, nil
}
