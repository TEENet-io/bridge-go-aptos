package database

import (
	"database/sql"
	"sync"
)

// to cache prepared sql statement, which maps query string to stmt.
type StmtCache struct {
	db *sql.DB
	m  sync.Map
}

func NewStmtCache(db *sql.DB) *StmtCache {
	return &StmtCache{db: db}
}

// Wrapped [Prepare()] with a cache (type sync.Map)
func (sc *StmtCache) Prepare(query string) (*sql.Stmt, error) {
	cached, _ := sc.m.Load(query)
	if cached == nil {
		stmt, err := sc.db.Prepare(query)
		if err != nil {
			return nil, err
		}
		sc.m.Store(query, stmt)
		cached = stmt
	}
	return cached.(*sql.Stmt), nil
}

// Clear cached statement
func (sc *StmtCache) Clear() {
	sc.m.Range(func(k, v interface{}) bool {
		_ = v.(*sql.Stmt).Close()
		sc.m.Delete(k)
		return true
	})
}
