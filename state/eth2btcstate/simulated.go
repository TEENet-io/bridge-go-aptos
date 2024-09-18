package eth2btcstate

func NewSimState(channelSize, cacheSize int) (*State, error) {
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
		return st, nil
	}
}
