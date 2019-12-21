package minerpool

import (
	"fmt"
	"github.com/hacash/core/fields"
	"github.com/hacash/miner/message"
	"net"
	"sync/atomic"
	"time"
)

// 监听端口
func (p *MinerPool) startServerListen() error {
	port := p.Config.TcpListenPort
	listen := net.TCPAddr{IP: net.IPv4zero, Port: port, Zone: ""}
	server, err := net.ListenTCP("tcp", &listen)
	if err != nil {
		return err
	}

	fmt.Printf("[Miner Pool] Start server and listen on port %d.\n", port)

	go func() {
		for {
			conn, err := server.AcceptTCP()
			if err != nil {
				continue
			}
			go p.acceptConn(conn)
		}
	}()

	return nil
}

func (p *MinerPool) acceptConn(conn *net.TCPConn) {

	if p.currentTcpConnectingCount > int32(p.Config.TcpConnectMaxSize) {
		conn.Write([]byte("too_many_connect"))
		conn.Close() // 连接最大值
		return
	}

	atomic.AddInt32(&p.currentTcpConnectingCount, 1)
	defer func() {
		atomic.AddInt32(&p.currentTcpConnectingCount, -1) // 减法
	}()

	// 如果还没有挖区块，则返回关闭，隔一段时间再次连接
	if p.currentRealtimePeriod == nil {
		conn.Write([]byte("not_ready_yet"))
		conn.Close()
		return
	}
	curperiod := p.currentRealtimePeriod
	// create client
	client := NewClient(nil, conn, curperiod.targetBlock)

	go func() {
		<-time.Tick(time.Second * 17)
		if client.address == nil {
			conn.Close() // err end
		}
	}()

	// read msg
	segdata := make([]byte, 2048)

	for {
		rn, err := conn.Read(segdata)
		if err != nil {
			break
		}

		newbytes := make([]byte, rn)
		copy(newbytes, segdata[0:rn])
		//fmt.Println("MinerPool: rn, err := conn.Read(segdata)", segdata[0:rn])

		if rn == 21 && client.address == nil { // post address

			//fmt.Println(segdata[0:21])
			client.address = fields.Address(newbytes[0:21])
			account := p.loadAccountAndAddPeriodByAddress(client.address)
			//fmt.Println("account.activeClients.Add(client) // add")
			account.activeClients.Add(client) // add
			client.belongAccount = account    // set belong
			// send mining stuff
			client.belongAccount.realtimePeriod.sendMiningStuffMsg(client.conn)

		} else if rn == message.PowMasterMsgSize && client.belongAccount != nil {

			//fmt.Println( "message.PowMasterMsgSize", segdata[0:rn] )

			powresult := message.NewPowMasterMsg()
			powresult.Parse(newbytes, 0)
			client.postPowResult(powresult) // return pow results

		}
	}

	// end
	//fmt.Println("client.belongAccount.activeClients.Remove(client)")
	client.belongAccount.activeClients.Remove(client) // remove
	conn.Close()

}
