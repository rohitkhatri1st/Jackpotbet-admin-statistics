package main

import (
	"admin-stats/server"
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := server.NewServer()
	s.StartServer()
	s.StartCron(ctx)

	err := sendSystemdNotification()
	if err != nil {
		s.ForceLog.Error(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-c

	s.StopServer()
	s.ForceLog.Info("Server stopped gracefully")
}

func sendSystemdNotification() error {
	notifySocket := os.Getenv("NOTIFY_SOCKET")
	if notifySocket != "" {
		state := "READY=1"
		socketAddr := &net.UnixAddr{
			Name: notifySocket,
			Net:  "unixgram",
		}
		conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
		if err != nil {
			return err
		}
		defer conn.Close()
		_, err = conn.Write([]byte(state))
		return err
	}
	return nil
}
