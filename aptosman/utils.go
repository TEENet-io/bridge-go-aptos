package aptosman

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
)

// StringToPrivateKey 将字符串转换为Ed25519私钥
func StringToPrivateKey(s string) (ed25519.PrivateKey, error) {
	s = trimPrefix(s, "0x")
	return hex.DecodeString(s)
}

func privateKeyToHex(privateKey ed25519.PrivateKey) string {
	return hex.EncodeToString(privateKey)
}

// 去除前缀
func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// GenPrivateKey 生成随机Ed25519私钥
func GenPrivateKey() ed25519.PrivateKey {
	// 使用标准库的方式生成私钥，确保大小正确
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("生成Ed25519私钥失败: %v", err))
	}
	
	// 确保私钥大小正确 (应该是32字节的种子)
	if len(privateKey) != ed25519.PrivateKeySize {
		panic(fmt.Sprintf("生成的Ed25519私钥大小不正确: %d", len(privateKey)))
	}
	
	// 验证公钥是否正确
	if len(publicKey) != ed25519.PublicKeySize {
		panic(fmt.Sprintf("生成的Ed25519公钥大小不正确: %d", len(publicKey)))
	}
	
	return privateKey
}

// GenPrivateKeys 生成多个随机私钥
func GenPrivateKeys(number int) []ed25519.PrivateKey {
	privateKeys := make([]ed25519.PrivateKey, number)
	for i := 0; i < number; i++ {
		privateKeys[i] = GenPrivateKey()
	}
	return privateKeys
}
// NewAccount 从私钥创建Aptos账户
func NewAccount(privateKey ed25519.PrivateKey) (*aptos.Account, error) {
	// 确保私钥大小正确
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("无效的Ed25519私钥大小: %d, 应为: %d", len(privateKey), ed25519.PrivateKeySize)
	}
	
	// 只使用私钥的种子部分(前32字节)
	seed := privateKey[:32]
	key := crypto.Ed25519PrivateKey{}
	err := key.FromBytes(seed)
	if err != nil {
		return nil, fmt.Errorf("从私钥创建Ed25519PrivateKey失败: %v", err)
	}
	account, err := aptos.NewAccountFromSigner(&key)
	if err != nil {
		return nil, fmt.Errorf("从签名者创建账户失败: %v", err)
	}
	return account, nil

}

// NewAccounts 从多个私钥创建多个账户
func NewAccounts(privateKeys []ed25519.PrivateKey) ([]*aptos.Account, error) {
	accounts := make([]*aptos.Account, len(privateKeys))
	for i, privateKey := range privateKeys {
		account, err := NewAccount(privateKey)
		if err != nil {
			return nil, err
		}
		accounts[i] = account
	}
	return accounts, nil
}

// Create account from private key
func createAccountFromPrivateKey(privateKeyHex string) (*aptos.Account, error) {
	// Decode private key
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse private key: %v", err)
	}

	// Create Ed25519 private key
	key := crypto.Ed25519PrivateKey{}
	err = key.FromBytes(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Ed25519 private key: %v", err)
	}

	// Create account from signer
	account, err := aptos.NewAccountFromSigner(&key)
	if err != nil {
		return nil, fmt.Errorf("Failed to create account from private key: %v", err)
	}

	return account, nil
}