package ipc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
)

var E_content_length_incorrect = errors.New("ipc内容长度不符")
var E_invalid_header = errors.New("ipc内容头部无效")

type Datagram []byte

//需要处理tcp粘包问题
func Receive(conn net.Conn) ([]Datagram, net.Addr, error) {

	result := make([]Datagram, 0)

	var received []byte

	clientAddr := conn.RemoteAddr()

	var length int
	var err error

	for {
		buff := make([]byte, 1024)
		length, err = conn.Read(buff)
		if err != nil {
			return nil, clientAddr, err
		}

		if length == 0 {
			return nil, clientAddr, io.EOF
		}

		if received == nil {
			received = make([]byte, length)
			copy(received, buff[:length])
		} else {
			received = append(received, buff[:length]...)
		}

		for {
			var datagram Datagram
			datagram, received, err = extractDatagram(received)
			if err != nil {
				return nil, clientAddr, err
			}
			if datagram != nil {
				result = append(result, datagram)
			} else {
				break
			}
		}

		if len(result) > 0 && received == nil {
			break
		}
	}

	return result, clientAddr, nil
}

func extractDatagram(buffer []byte) (Datagram, []byte, error) {

	if len(buffer) == 0 {
		return nil, nil, nil
	}

	var input Datagram

	input = make([]byte, len(buffer))
	copy(input, buffer)

	index := bytes.IndexAny(input, "#")
	if index != 0 {
		log.Println("error 开头不是#", index, string(buffer))
		return nil, nil, E_invalid_header
	}
	input = bytes.TrimPrefix(input, []byte("#"))
	index = bytes.IndexAny(input, "#")
	if index <= 0 {
		return nil, buffer, nil
	}
	contentLength, err := strconv.Atoi(string(input[:index]))
	if err != nil || contentLength == 0 {
		log.Println("error 不是数字", err, contentLength)
		return nil, nil, E_invalid_header
	}
	input = bytes.TrimPrefix(input[index:], []byte("#"))

	if contentLength == len(input) {
		return input, nil, nil
	} else if contentLength < len(input) {
		return input[:contentLength], input[contentLength:], nil
	} else {
		return nil, buffer, nil
	}
}

func Write(conn net.Conn, data []byte) error {

	contentLength := len(data)

	_, err := conn.Write(append([]byte(fmt.Sprintf("#%d#", contentLength)), data...))
	if err != nil {
		log.Println("error send: ", err)
		return err
	}

	return nil
}
