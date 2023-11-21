package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

func main() {
	fmt.Println("hello world!")

	server, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("server license err:", err)
		return
	}

	for {
		conn, err := server.Accept()
		if err != nil {
			fmt.Println("Accept failed:", err)
			continue
		}
		//go handleConn(conn)
		go sockts5(conn)
	}
}

func sockts5(conn net.Conn) {
	if err := socket5Auth(conn); err != nil {
		fmt.Println("socket5 Auth err:", err)
		conn.Close()
		return
	}

	target, err := socket5Connect(conn)
	if err != nil {
		fmt.Println("socket5 connect err:", err)
		conn.Close()
		return
	}

	socket5ForWard(conn, target)

}

//认证
func socket5Auth(conn net.Conn) error {
	buf := make([]byte, 256)

	//读取VER and NMETHODS
	n, err := io.ReadFull(conn, buf[:2])
	if n != 2 {
		return errors.New("read header:" + err.Error())
	}
	ver, nmethods := int(buf[0]), int(buf[1])
	if ver != 5 {
		return errors.New("invaild sockets version")
	}

	n, err = io.ReadFull(conn, buf[:nmethods])
	if n != nmethods {
		return errors.New("read methods:" + err.Error())
	}

	n, err = conn.Write([]byte{0x05, 0x00})
	if n != 2 || err != nil {
		return errors.New("write resp err:" + err.Error())
	}

	return nil
}

//建立连接
func socket5Connect(conn net.Conn) (net.Conn, error) {
	buf := make([]byte, 256)
	n, err := io.ReadFull(conn, buf[:4])
	if n != 4 {
		return nil, errors.New("read header :" + err.Error())
	}

	ver, cmd, _, typ := buf[0], buf[1], buf[2], buf[3]
	if ver != 5 || cmd != 1 {
		return nil, errors.New("invaild ver/cmd")
	}

	addr := ""
	switch typ {
	case 1:

		n, err = io.ReadFull(conn, buf[:4])
		if n != 4 {
			return nil, errors.New("invaild ipv4:" + err.Error())
		}
		addr = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
		fmt.Println("ipv4:", addr)
	case 3:
		n, err = io.ReadFull(conn, buf[:1])
		if n != 1 {
			return nil, errors.New("invaild hostname:" + err.Error())
		}
		addrLen := buf[0]
		n, err = io.ReadFull(conn, buf[:addrLen])
		if n != int(addrLen) {
			return nil, errors.New("invaild hostname:" + err.Error())
		}
		addr = string(buf[:addrLen])
		fmt.Println("hostname:", addr)
	case 4:
		return nil, errors.New("invaild ipv6 yet")

	default:
		return nil, errors.New("invaild typ")
	}

	n, err = io.ReadFull(conn, buf[:2])
	if n != 2 {
		return nil, errors.New("read port :" + err.Error())
	}

	port := binary.BigEndian.Uint16(buf[:2])

	destAddr := fmt.Sprintf("%s:%d", addr, port)
	fmt.Println("destaddr:", destAddr)
	target, err := net.Dial("tcp", destAddr)
	if err != nil {
		return nil, errors.New("net dial err:" + err.Error())
	}

	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		target.Close()
		return nil, errors.New("write resp err:" + err.Error())
	}
	return target, nil
}

//转发数据
func socket5ForWard(client, target net.Conn) {
	forward := func(src, dst net.Conn) {
		defer src.Close()
		defer dst.Close()
		io.Copy(src, dst)
	}

	go forward(client, target)
	go forward(target, client)

}

func handleConn(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	fmt.Println("client from ", remote)
	conn.Write([]byte("hello world"))
	conn.Close()
}
