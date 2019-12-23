package main

import (
	"fmt"
	"github.com/hacash/core/sys"
	"github.com/hacash/miner/console"
	"github.com/hacash/miner/memtxpool"
	"github.com/hacash/miner/miner"
	"github.com/hacash/miner/minerpool"
	"github.com/hacash/node/backend"
	rpc "github.com/hacash/service/deprecated"
	"os"
	"os/signal"
	"time"
)

func main() {

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	target_ini_file := "hacash_config.ini"
	//target_ini_file := "/home/shiqiujie/Desktop/Hacash/go/src/github.com/hacash/miner/run/minerpool/test.ini"
	//target_ini_file := ""
	if len(os.Args) >= 2 {
		target_ini_file = os.Args[1]
	}

	target_ini_file = sys.AbsDir(target_ini_file)

	if target_ini_file != "" {
		fmt.Println("Load ini config file: \"" + target_ini_file + "\" at time:" + time.Now().Format("01/02 15:04:05"))
	}

	hinicnf, err := sys.LoadInicnf(target_ini_file)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}

	//test_data_dir := "/home/shiqiujie/Desktop/Hacash/go/src/github.com/hacash/miner/run/minerpool/testdata"
	//hinicnf.SetMustDataDir(test_data_dir)

	hcnf := backend.NewBackendConfig(hinicnf)
	hnode, err := backend.NewBackend(hcnf)
	if err != nil {
		fmt.Println(err)
		return
	}
	blockchainobj := hnode.BlockChain()

	// start
	hnode.Start()

	txpool := memtxpool.NewMemTxPool(0, 1024*1024*50)
	txpool.SetBlockChain(blockchainobj)

	// hnode set tx pool
	hnode.SetTxPool(txpool)

	//lccnf := localcpu.NewPowWrapConfig(hinicnf)
	//powwrap := localcpu.NewPowWrap(lccnf)

	mpcnf := minerpool.NewMinerPoolConfig(hinicnf)
	miner_pool := minerpool.NewMinerPool(mpcnf)
	miner_pool.SetBlockChain(blockchainobj)
	miner_pool.SetTxPool(txpool)

	mcnf := miner.NewMinerConfig(hinicnf)
	miner := miner.NewMiner(mcnf)

	miner.SetBlockChain(blockchainobj)
	miner.SetTxPool(txpool)
	miner.SetPowServer(miner_pool)

	// check reward address and password
	if !mcnf.Rewards.Equal(mpcnf.RewardAccount.Address) {
		fmt.Println("[Config Error] miner rewards address must equal to miner pool rewards passward address.")
		fmt.Printf("[配置错误] 矿池自动发送奖励的地址的密码应该是地址 %s 而不是地址 %s 的密码。\n", mcnf.Rewards.ToReadable(), mpcnf.RewardAccount.AddressReadable)
		os.Exit(0)
	}

	miner_pool.Start()
	miner.Start()

	cscnf := console.NewMinerConsoleConfig(hinicnf)
	console_service := console.NewMinerConsole(cscnf)
	console_service.SetMiningPool(miner_pool)

	console_service.Start() // http service

	// http api service
	svcnf := rpc.NewDeprecatedApiServiceConfig(hinicnf)
	deprecatedApi := rpc.NewDeprecatedApiService(svcnf)
	deprecatedApi.SetBlockChain(blockchainobj)
	deprecatedApi.SetTxPool(txpool)
	deprecatedApi.SetBackend(hnode)
	deprecatedApi.Start()

	// do mining
	miner.StartMining()

	// download block datas
	//hnode.DownloadBlocksDataFromWebSocketApi("ws://127.0.0.1:3338/websocket", 1)

	s := <-c
	fmt.Println("Got signal:", s)

}
