package chaintxmgrdb

func (db *SQLiteChainTxMgrDB) Close() {
	_ = db.db.Close()
}
