package maria

import (
	"database/sql"
	"fmt"
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
	rows, err := Database.Query(query, values...)
	var scan []string
	if err != nil {
		fmt.Println(err)
		return []string{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var tmp string
		err := rows.Scan(&tmp)
		if err != nil {
			fmt.Println("[SafeQuery]:", err)
			continue
		}
		scan = append(scan, tmp)
	}
	return scan, nil
}
