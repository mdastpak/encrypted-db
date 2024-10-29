package models

import (
	"encrypted-db/internal/db"
)

// FetchDataFromPostgres fetches data from PostgreSQL and returns a list of results
func FetchDataFromPostgres(pgService *db.PostgresService) ([]map[string]interface{}, error) {
	rows, err := pgService.DB.Query("SELECT id, name, description FROM my_table")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int
		var name, description string
		if err := rows.Scan(&id, &name, &description); err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description,
		})
	}
	return results, nil
}
