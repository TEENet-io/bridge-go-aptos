package utils

func SatoshiToBtc(satoshi int64) float64 {
	return float64(satoshi) / 100000000
}

func BtcToSatoshi(btc float64) int64 {
	return int64(btc * 100000000)
}
