package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/labstack/gommon/log"

	"github.com/sKudryashov/stacksrv/internal/conn"
	connPkg "github.com/sKudryashov/stacksrv/internal/conn"
	"github.com/sKudryashov/stacksrv/internal/handler"
	"github.com/sKudryashov/stacksrv/pkg/logger"
)

func main() {
	// go turnOnProf()
	// defer profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop()
	var addr, addrCtrl string
	flag.StringVar(&addr, "service", ":8080", "service address endpoint")
	flag.StringVar(&addrCtrl, "control", ":8081", "service address endpoint")
	flag.Parse()
	stopCh := make(chan interface{}, 5)
	stoppedCh := make(chan interface{})
	restartCh := make(chan interface{})

	var srv *Server
	srv = NewServer(addr)

	go startControl(addrCtrl, restartCh)
	go srv.start(stopCh, stoppedCh)

	for {
		select {
		case <-restartCh:
			logger.App.Info("server restart signal received")
			stopCh <- struct{}{}
			logger.App.Info("issued stop signal for the server")
			srv.lstnr.Close()
			select {
			case <-stoppedCh:
				logger.App.Info("signal server stopped received, ready to restart .. ")
				srv = NewServer(addr)
				logger.App.Debug("new srv instance created")
				go srv.start(stopCh, stoppedCh)
			}
		}
	}
}

// Start with start you may either start or restart the server
func (srv *Server) start(stopCh <-chan interface{}, stoppedCh chan<- interface{}) {
	readingQueue := make(chan *conn.Conn, conn.MaxConn)
	stopWorkersCh := make(chan interface{})
	pool := conn.NewConnPool(stopWorkersCh)
	tcpHandler := handler.NewTCP(pool)
	logger.App.Infof("server started on address %s", srv.laddr)

	go tcpHandler.ConnListener(readingQueue, stopWorkersCh)
	for {
		select {
		case <-stopCh:
			srv.lstnr.Close()
			logger.App.Info("server stop signal received")
			close(stopWorkersCh)
			readingQueue = nil
			logger.App.Info("closing listeners.. ")
			//let every worker get its signals
			time.Sleep(1 * time.Second)
			stoppedCh <- struct{}{}
			logger.App.Info("server is stopped")
			return
		default:
		}
		conn, err := srv.lstnr.AcceptTCP()
		if err != nil {
			logger.App.Errorf("failed to accept conn: %", err)
			if conn != nil {
				conn.Close()
			}
			continue
		}
		logger.App.Infof("accepted tcp from ", conn.RemoteAddr())
		appConn := &connPkg.Conn{
			TCPConn: conn,
		}
		appConn.SetTime(time.Now().Unix())
		appConn.SetActive(true)
		appConn.SetNoDelay(true)
		pool.TryPush(appConn, readingQueue)
	}
}

func (srv *Server) stop() {
	if err := srv.lstnr.Close(); err != nil {
		panic(" unable to close 1" + err.Error())
	}
	file, err := srv.lstnr.File()
	if err == nil {
		file.Close()
	} else {
		panic(" unable to close 2" + err.Error())
	}
}

// NewServer is a server constructor
func NewServer(addr string) *Server {
	resolvedTCPAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		fmt.Println("resolvedTCPAddr error ", err.Error())
		os.Exit(1)
	}
	listener, err := net.ListenTCP("tcp", resolvedTCPAddr)
	if err != nil {
		fmt.Println("launch listener error ", err.Error())
		os.Exit(1)
	}

	return &Server{
		lstnr:   listener,
		tcpAddr: resolvedTCPAddr,
		laddr:   addr,
	}
}

func startControl(addrCtrl string, restartCh chan<- interface{}) {
	laddr, err := net.ResolveTCPAddr("tcp", addrCtrl)
	l, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		fmt.Println("launch error ", err.Error())
		os.Exit(1)
	}
	for {
		logger.Control.Infof("ready to accept control commands on addr %s", addrCtrl)
		conn, err := l.AcceptTCP()
		if err != nil {
			logger.Control.Errorf("failed to accept conn: %", err)
			conn.Close()
			continue
		}
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		conn.SetKeepAlive(false)
		data := make([]byte, 3)
		if _, err := conn.Read(data); err != nil {
			logger.Control.Errorf("unable to read conn %v", err)
		}
		conn.Close()
		logger.Control.Infof("accepted tcp from ", conn.RemoteAddr())
		if string(data) == "rel" {
			restartCh <- struct{}{}
		}
	}
}

type Server struct {
	lstnr   *net.TCPListener
	tcpAddr *net.TCPAddr
	laddr   string
}

func getLogLVL() log.Lvl {
	lvl := os.Getenv("LOG_LEVEL")
	switch lvl {
	case "info":
		return log.INFO
	case "error":
		return log.ERROR
	default:
		return log.DEBUG
	}
}

// func turnOnProf() {
// 	r := http.NewServeMux()
// 	runtime.SetBlockProfileRate(1)
// 	r.HandleFunc("/debug/pprof/", pprof.Index)
// 	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
// 	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
// 	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
// 	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

// 	http.ListenAndServe(":8090", r)
// }
