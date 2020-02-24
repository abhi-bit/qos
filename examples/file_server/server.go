package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/abhi-bit/qos"
	"go.uber.org/zap"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger

	seededRand *rand.Rand
)

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger, err: %s\n", err))
	}

	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func stringWithCharset(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func startServer() {
	listener, err := net.Listen("tcp", ":10000")
	if err != nil {
		sugar.Fatalf("failed to start up file server, err: %s", err)
	}
	sugar.Infof("Server listening on port 10000")

	qs := qos.WithDefaultConfig()
	for {
		conn, err := listener.Accept()
		if err != nil {
			sugar.Errorf("failed to accept incoming client request, err: %s", err)
			continue
		}

		go writeFileContents(qs, conn)
	}
}

func writeFileContents(qs *qos.QOS, conn net.Conn) error {
	defer conn.Close()

	fakeFileContents := []byte(stringWithCharset(4096))

	for i := 0; i < 1024*1024; i++ {
		if qs.Allowed(conn) {
			if n, err := conn.Write(fakeFileContents); err != nil {
				return err
			} else {
				qs.TrackConn(conn, uint64(n))
			}
		}
	}

	return nil
}

func main() {
	defer logger.Sync()
	sugar = logger.Sugar()
	startServer()
}
