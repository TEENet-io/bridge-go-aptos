package eth2btcstate

import "database/sql"

type SimState struct {
	*State
}

func NewSimState(sqldb *sql.DB, channelSize int) (*SimState, error) {
	statedb, err := NewStateDB(sqldb)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ChannelSize: channelSize,
	}

	if st, err := New(statedb, cfg); err != nil {
		return nil, err
	} else {
		return &SimState{st}, nil
	}
}

func (st *SimState) Close() {
	st.State.Close()
	st.db.Close()
}
