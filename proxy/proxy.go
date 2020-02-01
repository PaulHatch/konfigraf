package proxy

// Database definition
type DB struct {
	ExecFunc     func(query string, types []string, args []interface{}) error
	QueryFunc    func(query string, types []string, args []interface{}) (*Rows, error)
	QueryRowFunc func(query string, types []string, args []interface{}) (*Row, error)
}

func (db *DB) Exec(query string, types []string, args ...interface{}) error {
	return db.ExecFunc(query, types, args)
}

func (db *DB) Query(query string, types []string, args ...interface{}) (*Rows, error) {
	return db.QueryFunc(query, types, args)
}

func (db *DB) QueryRow(query string, types []string, args ...interface{}) (*Row, error) {
	return db.QueryRowFunc(query, types, args)
}

// Rows from a query
type Rows struct {
	NextFunc func() bool
	ScanFunc func(args []interface{}) error
}

func (rows *Rows) Next() bool {
	return rows.NextFunc()
}

func (rows *Rows) Scan(args ...interface{}) error {
	return rows.ScanFunc(args)
}

// Row from a query
type Row struct {
	ScanFunc func(args []interface{}) error
}

func (row *Row) Scan(args ...interface{}) error {
	return row.ScanFunc(args)
}
