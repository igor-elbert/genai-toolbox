// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snowflake

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
)

// ExecuteQuery executes a SQL query on the Snowflake database and returns the results.
func ExecuteQuery(ctx context.Context, db *sqlx.DB, query string, params ...interface{}) ([]any, error) {
	rows, err := db.QueryxContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Snowflake query: %w", err)
	}
	defer rows.Close()

	var out []any
	for rows.Next() {
		cols, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("failed to get columns from Snowflake result: %w", err)
		}

		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row from Snowflake result: %w", err)
		}

		vMap := make(map[string]any)
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				vMap[col] = string(b)
			} else {
				vMap[col] = val
			}
		}
		out = append(out, vMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("an error occurred during row iteration: %w", err)
	}

	return out, nil
}
