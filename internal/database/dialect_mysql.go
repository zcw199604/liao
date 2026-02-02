package database

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

// MySQLDialect implements Dialect for MySQL.
type MySQLDialect struct{}

func (MySQLDialect) Name() string { return "mysql" }

func (MySQLDialect) DriverName() string { return "mysql" }

func (MySQLDialect) Rebind(query string) string { return query }

func (MySQLDialect) IsDuplicateKey(err error) bool {
	var myErr *mysql.MySQLError
	// 1062: ER_DUP_ENTRY
	return errors.As(err, &myErr) && myErr != nil && myErr.Number == 1062
}

func (MySQLDialect) IsDuplicateColumn(err error) bool {
	var myErr *mysql.MySQLError
	// 1060: ER_DUP_FIELDNAME
	return errors.As(err, &myErr) && myErr != nil && myErr.Number == 1060
}

func (MySQLDialect) IsDuplicateIndex(err error) bool {
	var myErr *mysql.MySQLError
	// 1061: ER_DUP_KEYNAME
	return errors.As(err, &myErr) && myErr != nil && myErr.Number == 1061
}
