package shm

import (
	"net"
	"encoding/binary"
	"io"
)

var COMMON_SOCK = "/tmp/uds-common-shm.sock"
var COMMON_SHM_SIZE = 1<<25

func ReadInt(fd *net.Conn) (uint32, error) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(*fd, buf)
	if err != nil {
		return 0, err
	}
	num := binary.LittleEndian.Uint32(buf)
	return num, nil
}

func WriteInt(fd *net.Conn, num uint32) error {
	respBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(respBytes, num)
	_, err := (*fd).Write(respBytes)
	return err
}

