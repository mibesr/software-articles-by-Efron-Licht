package main

import (
	"bufio"
	"fmt"
	"net"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func echo(logger *zap.Logger, port int) error {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: port})
	if err != nil {
		return fmt.Errorf("listen tcp on port %d: %w", port, err)
	}
	logger = logger.With(zap.Object("echo", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("addr", listener.Addr().String())
		enc.AddInt("port", port)
		return nil
	})))

	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept connection: %w", err)
		}
		go func() {
			defer conn.Close()
			for scanner := bufio.NewScanner(conn); scanner.Scan(); {
				line := scanner.Bytes()
				if _, err := fmt.Fprintf(conn, "%s\n", line); err != nil {
					logger.Error("error writing to connection", zap.Error(err))

				}
				logger.Info("got line", zap.ByteString("line", line))
				if _, err := conn.Write(append(line, '\n')); err != nil {
					logger.Error("error writing to connection", zap.Error(err))
				}
			}
		}()
	}

}
