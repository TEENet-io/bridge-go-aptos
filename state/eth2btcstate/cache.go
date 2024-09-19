package eth2btcstate

import (
	"github.com/ethereum/go-ethereum/common/lru"
)

type redeemCacheKey struct {
	txHash [32]byte
	status RedeemStatus
}

type redeemCache struct {
	cache *lru.Cache[redeemCacheKey, *Redeem]
}

func newRedeemCache(size int) *redeemCache {
	return &redeemCache{
		cache: lru.NewCache[redeemCacheKey, *Redeem](size),
	}
}

func (rc *redeemCache) add(r *Redeem) {
	rc.cache.Add(redeemCacheKey{r.RequestTxHash, r.Status}, r)
}

func (rc *redeemCache) get(txHash [32]byte, status RedeemStatus) (*Redeem, bool) {
	return rc.cache.Get(redeemCacheKey{txHash, status})
}

func (rc *redeemCache) remove(txHash [32]byte, status RedeemStatus) bool {
	return rc.cache.Remove(redeemCacheKey{txHash, status})
}
