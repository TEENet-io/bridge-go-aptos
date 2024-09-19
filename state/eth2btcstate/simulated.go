package eth2btcstate

type SimState struct {
	*State
}

func NewSimState(channelSize, cacheSize int) (*SimState, error) {
	db, err := newStateDB("sqlite3", ":memory:")
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
