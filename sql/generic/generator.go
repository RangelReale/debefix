package generic

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	debefix_poc2 "github.com/RangelReale/debefix-poc2"
)

type ResolverDBCallback func(tableName string, fields map[string]any, returnFieldNames []string) (map[string]any, error)

func ResolverFunc(callback ResolverDBCallback) func(ctx debefix_poc2.ResolveContext, fields map[string]any) error {
	return func(ctx debefix_poc2.ResolveContext, fields map[string]any) error {
		insertFields := map[string]any{}
		var returnFieldNames []string

		for fn, fv := range fields {
			if fresolve, ok := fv.(debefix_poc2.ResolveValue); ok {
				switch fresolve.(type) {
				case *debefix_poc2.ResolveGenerate:
					returnFieldNames = append(returnFieldNames, fn)
				}
			} else {
				insertFields[fn] = fv
			}
		}

		resolved, err := callback(ctx.TableName(), insertFields, returnFieldNames)
		if err != nil {
			return err
		}

		for rn, rv := range resolved {
			ctx.ResolveField(rn, rv)
		}

		return nil
	}
}

type SQLPlaceholderProvider interface {
	Next() (placeholder string, argName string)
}

type SQLBuilder interface {
	CreatePlaceholderProvider() SQLPlaceholderProvider
	BuildInsertSQL(tableName string, fieldNames []string, fieldPlaceholders []string, returnFieldNames []string) string
}

func SQLResolverDBCallback(db QueryInterface, sqlBuilder SQLBuilder) ResolverDBCallback {
	return func(tableName string, fields map[string]any, returnFieldNames []string) (map[string]any, error) {
		var fieldNames []string
		var fieldPlaceholders []string
		var args []any

		placeholderProvider := sqlBuilder.CreatePlaceholderProvider()

		for fn, fv := range fields {
			fieldNames = append(fieldNames, fn)
			placeholder, argName := placeholderProvider.Next()
			fieldPlaceholders = append(fieldPlaceholders, placeholder)
			if argName != "" {
				args = append(args, sql.Named(argName, fv))
			} else {
				args = append(args, fv)
			}
		}

		query := sqlBuilder.BuildInsertSQL(tableName, fieldNames, fieldPlaceholders, returnFieldNames)

		ret, err := db.Query(query, returnFieldNames, args...)
		if err != nil {
			return nil, err
		}

		return ret, nil
	}
}

type RowInterface interface {
	Scan(dest ...any) error
}

type QueryInterface interface {
	Query(query string, returnFieldNames []string, args ...any) (map[string]any, error)
}

func RowToMap(cols []string, row RowInterface) (map[string]any, error) {
	// Create a slice of interface{}'s to represent each column,
	// and a second slice to contain pointers to each item in the columns slice.
	columns := make([]interface{}, len(cols))
	columnPointers := make([]interface{}, len(cols))
	for i, _ := range columns {
		columnPointers[i] = &columns[i]
	}

	// Scan the result into the column pointers...
	if err := row.Scan(columnPointers...); err != nil {
		return nil, err
	}

	// Create our map, and retrieve the value for each column from the pointers slice,
	// storing it in the map with the name of the column as the key.
	m := make(map[string]interface{})
	for i, colName := range cols {
		val := columnPointers[i].(*interface{})
		m[colName] = *val
	}

	return m, nil
}

type defaultSQLPlaceholderProvider struct {
}

func (d defaultSQLPlaceholderProvider) Next() (placeholder string, argName string) {
	return "?", ""
}

type DefaultSQLBuilder struct {
	PlaceholderProviderFactory func() SQLPlaceholderProvider
	QuoteTable                 func(t string) string
	QuoteField                 func(f string) string
}

func (d DefaultSQLBuilder) CreatePlaceholderProvider() SQLPlaceholderProvider {
	if d.PlaceholderProviderFactory == nil {
		return &defaultSQLPlaceholderProvider{}
	}
	return d.PlaceholderProviderFactory()
}

func (d DefaultSQLBuilder) BuildInsertSQL(tableName string, fieldNames []string, fieldPlaceholders []string, returnFieldNames []string) string {
	if d.QuoteTable != nil {
		tableName = d.QuoteTable(tableName)
	}

	if d.QuoteField != nil {
		fieldNames = slices.Clone(fieldNames)
		for fi := range fieldNames {
			fieldNames[fi] = d.QuoteField(fieldNames[fi])
		}
	}

	ret := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(fieldNames, ", "),
		strings.Join(fieldPlaceholders, ", "),
	)

	if len(returnFieldNames) > 0 {
		ret += fmt.Sprintf(" RETURNING %s", strings.Join(returnFieldNames, ","))
	}
	return ret
}

type SQLQueryInterface struct {
	DB *sql.DB
}

func (q SQLQueryInterface) Query(query string, args ...any) (map[string]any, error) {
	rows, err := q.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, errors.New("no records")
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	ret, err := RowToMap(cols, rows)
	if err != nil {
		return nil, err
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return ret, nil
}
