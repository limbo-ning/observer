package engine_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"testing"
	"time"

	myContext "obsessiontech/common/context"
)

const readBuffSize = 1024

func receive(conn *net.TCPConn) (string, error) {
	var datagram []byte

	buf := make([]byte, readBuffSize)
	for {
		length, err := conn.Read(buf)
		if err == io.EOF {
			log.Println("read eof", conn.RemoteAddr().String())
			break
		} else if err != nil {
			log.Println("read error:", err)
			return "", err
		} else if length == 0 {
			log.Println("read length 0", conn.RemoteAddr().String())
		} else {
			if length < len(buf) {
				log.Println("length < buffer: ", length, string(buf))
				datagram = append(datagram, buf[0:length]...)
				if bytes.HasSuffix(buf[0:length], []byte("\n")) {
					log.Println("break: ", string(buf[0:length]))
					break
				}
			} else {
				datagram = append(datagram, buf...)
				buf = make([]byte, len(buf)+readBuffSize)
				continue
			}
		}
	}

	if len(datagram) == 0 {
		log.Println("read error empty data:", io.ErrUnexpectedEOF, conn.RemoteAddr().String())
		return "", io.ErrUnexpectedEOF
	}

	str := string(datagram)
	log.Println("接收报文: ", len(datagram), str)

	return str, nil
}

func startHost(test func(*net.TCPConn, func())) {
	tcpAddress, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf(":%d", 6789))
	if err != nil {
		log.Panic(err)
	}

	tcp, err := net.ListenTCP("tcp4", tcpAddress)
	if err != nil {
		log.Panic(err)
	}

	log.Println("tcp listener started")

	ctx, cancel := myContext.GetContext()
	closeListener := func() {
		tcp.Close()
		cancel()
	}

	listener := make(chan *net.TCPConn)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("receiver listener stop")
				return
			default:
			}
			conn, err := tcp.AcceptTCP()
			if err != nil {
				log.Println("error accept", err)
				continue
			}

			listener <- conn
		}
	}()

	for {
		select {
		case conn := <-listener:
			test(conn, closeListener)
		case <-ctx.Done():
			log.Println("receiver listener closing")
			closeListener()
			//这里不要return中断主程序 用select阻塞 让context来执行gracefully exit 详见common/context
			select {}
		}
	}
}

func dialHost(remote string, input []byte) error {
	host, err := net.ResolveTCPAddr("tcp4", remote)
	if err != nil {
		log.Println("error dial: ", err)
		return err
	}
	tcp, err := net.DialTCP("tcp4", nil, host)
	if err != nil {
		log.Println("error dial: ", err)
		return err
	}

	defer tcp.Close()

	written, err := tcp.Write(input)
	if err != nil {
		log.Println("error write: ", err)
		return err
	}

	log.Println("sent: ", written)
	return nil
}

func TestReceiveBuffer1(t *testing.T) {
	go startHost(func(conn *net.TCPConn, close func()) {

		defer close()

		received, err := receive(conn)
		if err != nil {
			log.Println("error : ", err)
		}

		log.Println("received: ", received)
	})

	time.Sleep(time.Second * 5)

	// dialHost([]byte("##0813QN=20210722140000509;ST=31;CN=2061;PW=123456;MN=NEWPROTOCOLTEST000000092;Flag=4;CP=&&DataTime=20210722130000;a24088-Min=0.0000,a24088-Avg=0.0196,a24088-Max=0.1988,a24088-Cou=0.0003,a24088-Flag=N;a01012-Min=38.9250,a01012-Avg=39.7515,a01012-Max=40.7387,a01012-Flag=T;a01013-Min=-130.7116,a01013-Avg=-123.0723,a01013-Max=-115.9674,a01013-Flag=T;a01011-Min=5.4136,a01011-Avg=5.7286,a01011-Max=5.8780,a01011-Flag=T;a05002-Min=0.0110,a05002-Avg=0.1293,a05002-Max=1.1489,a05002-Cou=0.0017,a05002-Flag=N;a24087-Min=0.0110,a24087-Avg=0.1489,a24087-Max=1.3478,a24087-Cou=0.0019,a24087-Flag=N;a01014-Min=9.4451,a01014-Avg=9.6075,a01014-Max=9.9131,a01014-Flag=T;a19001-Min=15.1661,a19001-Avg=15.1909,a19001-Max=15.2188,a19001-Flag=T;a00000-Min=3.7077,a00000-Avg=3.9151,a00000-Max=4.0276,a00000-Cou=12758.2462,a00000-Flag=N&&5140"))

	dialHost("127.0.0.1:6789", []byte("##0813QN=20210722140000509;ST=31;CN=2061;PW=123456;MN=NEWPROTOCOLTEST000000092;Flag=4;CP=&&DataTime=20210722130000;a24088-Min=0.0000,a24088-Avg=0.0196,a24088-Max=0.1988,a24088-Cou=0.0003,a24088-Flag=N;a01012-Min=38.9250,a01012-Avg=39.7515,a01012-Max=40.7387,a01012-Flag=T;a01013-Min=-130.7116,a01013-Avg=-123.0723,a01013-Max=-115.9674,a01013-Flag=T;a01011-Min=5.4136,a01011-Avg=5.7286,a01011-Max=5.8780,a01011-Flag=T;a05002-Min=0.0110,a05002-Avg=0.1293,a05002-Max=1.1489,a05002-Cou=0.0017,a05002-Flag=N;a24087-Min=0.0110,a24087-Avg=0.1489,a24087-Max=1.3478,a24087-Cou=0.0019,a24087-Flag=N;a01014-Min=9.4451,a01014-Avg=9.6075,a01014-Max=9.9131,a01014-Flag=T;a19001-Min=15.1661,a19001-Avg=15.1909,a19001-Max=15.2188,a19001-Flag=T;a00000-Min=3.7077,a00000-Avg=3.9151,a00000-Max=4.0276,a00000-Cou=12758.2462,a00000-Flag=N&&5140##0813QN=20210722140000509;ST=31;CN=2061;PW=123456;MN=NEWPROTOCOLTEST000000092;Flag=4;CP=&&DataTime=20210722130000;a24088-Min=0.0000,a24088-Avg=0.0196,a24088-Max=0.1988,a24088-Cou=0.0003,a24088-Flag=N;a01012-Min=38.9250,a01012-Avg=39.7515,a01012-Max=40.7387,a01012-Flag=T;a01013-Min=-130.7116,a01013-Avg=-123.0723,a01013-Max=-115.9674,a01013-Flag=T;a01011-Min=5.4136,a01011-Avg=5.7286,a01011-Max=5.8780,a01011-Flag=T;a05002-Min=0.0110,a05002-Avg=0.1293,a05002-Max=1.1489,a05002-Cou=0.0017,a05002-Flag=N;a24087-Min=0.0110,a24087-Avg=0.1489,a24087-Max=1.3478,a24087-Cou=0.0019,a24087-Flag=N;a01014-Min=9.4451,a01014-Avg=9.6075,a01014-Max=9.9131,a01014-Flag=T;a19001-Min=15.1661,a19001-Avg=15.1909,a19001-Max=15.2188,a19001-Flag=T;a00000-Min=3.7077,a00000-Avg=3.9151,a00000-Max=4.0276,a00000-Cou=12758.2462,a00000-Flag=N&&5140##0813QN=20210722140000509;ST=31;CN=2061;PW=123456;MN=NEWPROTOCOLTEST000000092;Flag=4;CP=&&DataTime=20210722130000;a24088-Min=0.0000,a24088-Avg=0.0196,a24088-Max=0.1988,a24088-Cou=0.0003,a24088-Flag=N;a01012-Min=38.9250,a01012-Avg=39.7515,a01012-Max=40.7387,a01012-Flag=T;a01013-Min=-130.7116,a01013-Avg=-123.0723,a01013-Max=-115.9674,a01013-Flag=T;a01011-Min=5.4136,a01011-Avg=5.7286,a01011-Max=5.8780,a01011-Flag=T;a05002-Min=0.0110,a05002-Avg=0.1293,a05002-Max=1.1489,a05002-Cou=0.0017,a05002-Flag=N;a24087-Min=0.0110,a24087-Avg=0.1489,a24087-Max=1.3478,a24087-Cou=0.0019,a24087-Flag=N;a01014-Min=9.4451,a01014-Avg=9.6075,a01014-Max=9.9131,a01014-Flag=T;a19001-Min=15.1661,a19001-Avg=15.1909,a19001-Max=15.2188,a19001-Flag=T;a00000-Min=3.7077,a00000-Avg=3.9151,a00000-Max=4.0276,a00000-Cou=12758.2462,a00000-Flag=N&&5140##0813QN=20210722140000509;ST=31;CN=2061;PW=123456;MN=NEWPROTOCOLTEST000000092;Flag=4;CP=&&DataTime=20210722130000;a24088-Min=0.0000,a24088-Avg=0.0196,a24088-Max=0.1988,a24088-Cou=0.0003,a24088-Flag=N;a01012-Min=38.9250,a01012-Avg=39.7515,a01012-Max=40.7387,a01012-Flag=T;a01013-Min=-130.7116,a01013-Avg=-123.0723,a01013-Max=-115.9674,a01013-Flag=T;a01011-Min=5.4136,a01011-Avg=5.7286,a01011-Max=5.8780,a01011-Flag=T;a05002-Min=0.0110,a05002-Avg=0.1293,a05002-Max=1.1489,a05002-Cou=0.0017,a05002-Flag=N;a24087-Min=0.0110,a24087-Avg=0.1489,a24087-Max=1.3478,a24087-Cou=0.0019,a24087-Flag=N;a01014-Min=9.4451,a01014-Avg=9.6075,a01014-Max=9.9131,a01014-Flag=T;a19001-Min=15.1661,a19001-Avg=15.1909,a19001-Max=15.2188,a19001-Flag=T;a00000-Min=3.7077,a00000-Avg=3.9151,a00000-Max=4.0276,a00000-Cou=12758.2462,a00000-Flag=N&&5140\r\n"))

	time.Sleep(time.Second * 5)
}

func TestDialServer(t *testing.T) {
	dialHost("60.205.210.219:10020", []byte("##0893QN=20210722132000042;ST=31;CN=2051;PW=123456;MN=41082201S15303;Flag=4;CP=&&DataTime=20210722132000;a24088-Min=0.0002,a24088-Avg=0.0002,a24088-Max=0.0002,a24088-ZsMin=0.0002,a24088-ZsAvg=0.0002,a24088-ZsMax=0.0002,a24088-Cou=0.0000,a24088-Flag=N;a01012-Min=39.4953,a01012-Avg=39.4953,a01012-Max=39.4953,a01012-Flag=T;a01013-Min=-120.4117,a01013-Avg=-120.4117,a01013-Max=-120.4117,a01013-Flag=T;a01011-Min=5.8497,a01011-Avg=5.8497,a01011-Max=5.8497,a01011-Flag=T;a05002-Min=0.0221,a05002-Avg=0.0221,a05002-Max=0.0221,a05002-ZsMin=0.0221,a05002-ZsAvg=0.0221,a05002-ZsMax=0.0221,a05002-Cou=0.0000,a05002-Flag=N;a24087-Min=0.0223,a24087-Avg=0.0223,a24087-Max=0.0223,a24087-ZsMin=0.0223,a24087-ZsAvg=0.0223,a24087-ZsMax=0.0223,a24087-Cou=0.0000,a24087-Flag=N;a01014-Min=9.4608,a01014-Avg=9.4608,a01014-Max=9.4608,a01014-Flag=T;a19001-Min=15.1771,a19001-Avg=15.1771,a19001-Max=15.1771,a19001-Flag=T&&24C0\r\n##0893QN=20210722132000042;ST=31;CN=2051;PW=123456;MN=41082201S15303;Flag=4;CP=&&DataTime=20210722132000;a24088-Min=0.0002,a24088-Avg=0.0002,a24088-Max=0.0002,a24088-ZsMin=0.0002,a24088-ZsAvg=0.0002,a24088-ZsMax=0.0002,a24088-Cou=0.0000,a24088-Flag=N;a01012-Min=39.4953,a01012-Avg=39.4953,a01012-Max=39.4953,a01012-Flag=T;a01013-Min=-120.4117,a01013-Avg=-120.4117,a01013-Max=-120.4117,a01013-Flag=T;a01011-Min=5.8497,a01011-Avg=5.8497,a01011-Max=5.8497,a01011-Flag=T;a05002-Min=0.0221,a05002-Avg=0.0221,a05002-Max=0.0221,a05002-ZsMin=0.0221,a05002-ZsAvg=0.0221,a05002-ZsMax=0.0221,a05002-Cou=0.0000,a05002-Flag=N;a24087-Min=0.0223,a24087-Avg=0.0223,a24087-Max=0.0223,a24087-ZsMin=0.0223,a24087-ZsAvg=0.0223,a24087-ZsMax=0.0223,a24087-Cou=0.0000,a24087-Flag=N;a01014-Min=9.4608,a01014-Avg=9.4608,a01014-Max=9.4608,a01014-Flag=T;a19001-Min=15.1771,a19001-Avg=15.1771,a19001-Max=15.1771,a19001-Flag=T&&24C0\r\n##0893QN=20210722132000042;ST=31;CN=2051;PW=123456;MN=41082201S15303;Flag=4;CP=&&DataTime=20210722132000;a24088-Min=0.0002,a24088-Avg=0.0002,a24088-Max=0.0002,a24088-ZsMin=0.0002,a24088-ZsAvg=0.0002,a24088-ZsMax=0.0002,a24088-Cou=0.0000,a24088-Flag=N;a01012-Min=39.4953,a01012-Avg=39.4953,a01012-Max=39.4953,a01012-Flag=T;a01013-Min=-120.4117,a01013-Avg=-120.4117,a01013-Max=-120.4117,a01013-Flag=T;a01011-Min=5.8497,a01011-Avg=5.8497,a01011-Max=5.8497,a01011-Flag=T;a05002-Min=0.0221,a05002-Avg=0.0221,a05002-Max=0.0221,a05002-ZsMin=0.0221,a05002-ZsAvg=0.0221,a05002-ZsMax=0.0221,a05002-Cou=0.0000,a05002-Flag=N;a24087-Min=0.0223,a24087-Avg=0.0223,a24087-Max=0.0223,a24087-ZsMin=0.0223,a24087-ZsAvg=0.0223,a24087-ZsMax=0.0223,a24087-Cou=0.0000,a24087-Flag=N;a01014-Min=9.4608,a01014-Avg=9.4608,a01014-Max=9.4608,a01014-Flag=T;a19001-Min=15.1771,a19001-Avg=15.1771,a19001-Max=15.1771,a19001-Flag=T&&24C0\r\n"))
	// dialHost("60.205.210.219:10020", []byte("##0813QN=20210722140000509;ST=31;CN=2061;PW=123456;MN=NEWPROTOCOLTEST000000092;Flag=4;CP=&&DataTime=20210722130000;a24088-Min=0.0000,a24088-Avg=0.0196,a24088-Max=0.1988,a24088-Cou=0.0003,a24088-Flag=N;a01012-Min=38.9250,a01012-Avg=39.7515,a01012-Max=40.7387,a01012-Flag=T;a01013-Min=-130.7116,a01013-Avg=-123.0723,a01013-Max=-115.9674,a01013-Flag=T;a01011-Min=5.4136,a01011-Avg=5.7286,a01011-Max=5.8780,a01011-Flag=T;a05002-Min=0.0110,a05002-Avg=0.1293,a05002-Max=1.1489,a05002-Cou=0.0017,a05002-Flag=N;a24087-Min=0.0110,a24087-Avg=0.1489,a24087-Max=1.3478,a24087-Cou=0.0019,a24087-Flag=N;a01014-Min=9.4451,a01014-Avg=9.6075,a01014-Max=9.9131,a01014-Flag=T;a19001-Min=15.1661,a19001-Avg=15.1909,a19001-Max=15.2188,a19001-Flag=T;a00000-Min=3.7077,a00000-Avg=3.9151,a00000-Max=4.0276,a00000-Cou=12758.2462,a00000-Flag=N&&5140##0813QN=20210722140000509;ST=31;CN=2061;PW=123456;MN=NEWPROTOCOLTEST000000092;Flag=4;CP=&&DataTime=20210722130000;a24088-Min=0.0000,a24088-Avg=0.0196,a24088-Max=0.1988,a24088-Cou=0.0003,a24088-Flag=N;a01012-Min=38.9250,a01012-Avg=39.7515,a01012-Max=40.7387,a01012-Flag=T;a01013-Min=-130.7116,a01013-Avg=-123.0723,a01013-Max=-115.9674,a01013-Flag=T;a01011-Min=5.4136,a01011-Avg=5.7286,a01011-Max=5.8780,a01011-Flag=T;a05002-Min=0.0110,a05002-Avg=0.1293,a05002-Max=1.1489,a05002-Cou=0.0017,a05002-Flag=N;a24087-Min=0.0110,a24087-Avg=0.1489,a24087-Max=1.3478,a24087-Cou=0.0019,a24087-Flag=N;a01014-Min=9.4451,a01014-Avg=9.6075,a01014-Max=9.9131,a01014-Flag=T;a19001-Min=15.1661,a19001-Avg=15.1909,a19001-Max=15.2188,a19001-Flag=T;a00000-Min=3.7077,a00000-Avg=3.9151,a00000-Max=4.0276,a00000-Cou=12758.2462,a00000-Flag=N&&5140##0813QN=20210722140000509;ST=31;CN=2061;PW=123456;MN=NEWPROTOCOLTEST000000092;Flag=4;CP=&&DataTime=20210722130000;a24088-Min=0.0000,a24088-Avg=0.0196,a24088-Max=0.1988,a24088-Cou=0.0003,a24088-Flag=N;a01012-Min=38.9250,a01012-Avg=39.7515,a01012-Max=40.7387,a01012-Flag=T;a01013-Min=-130.7116,a01013-Avg=-123.0723,a01013-Max=-115.9674,a01013-Flag=T;a01011-Min=5.4136,a01011-Avg=5.7286,a01011-Max=5.8780,a01011-Flag=T;a05002-Min=0.0110,a05002-Avg=0.1293,a05002-Max=1.1489,a05002-Cou=0.0017,a05002-Flag=N;a24087-Min=0.0110,a24087-Avg=0.1489,a24087-Max=1.3478,a24087-Cou=0.0019,a24087-Flag=N;a01014-Min=9.4451,a01014-Avg=9.6075,a01014-Max=9.9131,a01014-Flag=T;a19001-Min=15.1661,a19001-Avg=15.1909,a19001-Max=15.2188,a19001-Flag=T;a00000-Min=3.7077,a00000-Avg=3.9151,a00000-Max=4.0276,a00000-Cou=12758.2462,a00000-Flag=N&&5140##0813QN=20210722140000509;ST=31;CN=2061;PW=123456;MN=NEWPROTOCOLTEST000000092;Flag=4;CP=&&DataTime=20210722130000;a24088-Min=0.0000,a24088-Avg=0.0196,a24088-Max=0.1988,a24088-Cou=0.0003,a24088-Flag=N;a01012-Min=38.9250,a01012-Avg=39.7515,a01012-Max=40.7387,a01012-Flag=T;a01013-Min=-130.7116,a01013-Avg=-123.0723,a01013-Max=-115.9674,a01013-Flag=T;a01011-Min=5.4136,a01011-Avg=5.7286,a01011-Max=5.8780,a01011-Flag=T;a05002-Min=0.0110,a05002-Avg=0.1293,a05002-Max=1.1489,a05002-Cou=0.0017,a05002-Flag=N;a24087-Min=0.0110,a24087-Avg=0.1489,a24087-Max=1.3478,a24087-Cou=0.0019,a24087-Flag=N;a01014-Min=9.4451,a01014-Avg=9.6075,a01014-Max=9.9131,a01014-Flag=T;a19001-Min=15.1661,a19001-Avg=15.1909,a19001-Max=15.2188,a19001-Flag=T;a00000-Min=3.7077,a00000-Avg=3.9151,a00000-Max=4.0276,a00000-Cou=12758.2462,a00000-Flag=N&&5140\r\n"))
}
