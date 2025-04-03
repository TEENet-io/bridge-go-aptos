package aptossync

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/aptosman"
	"github.com/TEENet-io/bridge-go/common"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// // MockAptosman 模拟Aptosman接口
type MockAptosman struct {
	mock.Mock
	Aptosman *aptosman.Aptosman
}

// func (m *MockAptosman) GetLatestFinalizedVersion() (uint64, error) {
// 	args := m.Called()
// 	return args.Get(0).(uint64), args.Error(1)
// }

func (m *MockAptosman) GetModuleEvents(fromVersion, toVersion uint64) (
	[]aptosman.MintedEvent,
	[]aptosman.RedeemRequestedEvent,
	[]aptosman.RedeemPreparedEvent,
	error,
) {
	args := m.Called(fromVersion, toVersion)
	return args.Get(0).([]aptosman.MintedEvent),
		args.Get(1).([]aptosman.RedeemRequestedEvent),
		args.Get(2).([]aptosman.RedeemPreparedEvent),
		args.Error(3)
}

func TestAptosSync(t *testing.T) {
	common.Debug = true
	defer func() {
		logger.Debug("DEBUG MODE OFF")
		common.Debug = false
	}()

	// 创建模拟状态
	st := NewMockState()
	// 创建模拟Aptosman
	mockAptosman := new(MockAptosman)
	mockAptosman_1, err := aptosman.NewSimAptosman_from_privateKey("0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d")
	if err != nil {
		t.Fatalf("failed to create mock aptosman: %v", err)
	}
	mockAptosman.Aptosman = mockAptosman_1.Aptosman

	// mockAptosman.aptosman =
	// 配置同步器
	cfg := &AptosSyncConfig{
		IntervalCheckBlockchain: 500 * time.Millisecond,
		BtcChainConfig:          common.MainNetParams(),
		AptosChainID:            big.NewInt(1), // Devnet 1
	}
	// 创建同步器
	synchronizer, err := New(mockAptosman.Aptosman, st, cfg)
	assert.NoError(t, err)
	synchronizer.lastFinalized = big.NewInt(100)
	// 测试场景1: 没有新的区块
	// mockAptosman.Aptosman.On("GetLatestFinalizedVersion").Return(uint64(100), nil).Once()
	ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Second)
	go st.Start(ctx1)
	go synchronizer.Loop(ctx1)
	time.Sleep(600 * time.Millisecond)
	cancel1()
	// // 验证没有事件被发送
	// assert.Empty(t, st.mintedEv)
	// assert.Empty(t, st.requestedEv)
	// assert.Empty(t, st.preparedEv)

	// 测试场景2: 有新的区块和事件
	mockAptosman.On("GetLatestFinalizedVersion").Return(uint64(105), nil).Once()
	// 模拟从版本101到105的事件
	mintedEvents := []aptosman.MintedEvent{
		{
			MintTxHash: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			BtcTxId:    "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			Receiver:   "0xuser1",
			Amount:     100,
		},
	}

	requestedEvents := []aptosman.RedeemRequestedEvent{
		{
			RequestTxHash: "0x2234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Requester:     "0xuser2",
			Receiver:      "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
			Amount:        50,
		},
		{
			RequestTxHash: "0x3234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Requester:     "0xuser3",
			Receiver:      "invalid_btc_address",
			Amount:        30,
		},
	}

	preparedEvents := []aptosman.RedeemPreparedEvent{
		{
			PrepareTxHash: "0x4234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			RequestTxHash: "0x2234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Requester:     "0xuser2",
			Receiver:      "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
			Amount:        50,
			OutpointTxIds: []string{"0x5234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},
			OutpointIdxs:  []uint16{0},
		},
	}

	mockAptosman.On("GetModuleEvents", uint64(101), uint64(105)).Return(
		mintedEvents, requestedEvents, preparedEvents, nil,
	).Once()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	go st.Start(ctx2)
	go synchronizer.Loop(ctx2)
	time.Sleep(600 * time.Millisecond)
	cancel2()

	// 验证事件被正确处理
	fmt.Println("Minted Events:", st.mintedEv)
	fmt.Println("Requested Events:", st.requestedEv)
	fmt.Println("Prepared Events:", st.preparedEv)
	fmt.Println("mintedEvents Events:", mintedEvents)
	fmt.Println("requestedEvents Events:", requestedEvents)
	fmt.Println("preparedEvents Events:", preparedEvents)
	fmt.Println("st.mintedEv:", st.mintedEv)
	fmt.Println("st.requestedEv:", st.requestedEv)
	fmt.Println("st.preparedEv:", st.preparedEv)
	fmt.Println("================================================")
	// 验证最后的版本号被更新
	lastVersion, err := st.GetBlockchainFinalizedBlockNumber()
	fmt.Println("lastVersion:", lastVersion)
	fmt.Println("err:", err)
}
