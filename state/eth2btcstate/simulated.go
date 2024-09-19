package eth2btcstate

import "database/sql"

type SimState struct {
	*State
}

func NewSimState(channelSize, cacheSize int) (*SimState, error) {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	db, err := NewStateDB(sqlDB)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ChannelSize: channelSize,
		CacheSize:   cacheSize,
	}

	if st, err := New(db, cfg); err != nil {
		return nil, err
	} else {
		return &SimState{st}, nil
	}
}

func (st *SimState) Close() {
	st.State.Close()
	st.db.Close()
}
