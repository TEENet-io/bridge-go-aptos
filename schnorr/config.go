package schnorr

type Config struct {
	T int // threshold
	N int // total number of secret shares

	ChannelSize int
}
