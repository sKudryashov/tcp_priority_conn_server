package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/pkg/profile"
	"github.com/sKudryashov/stacksrv/internal/conn"
	connPkg "github.com/sKudryashov/stacksrv/internal/conn"
	"github.com/sKudryashov/stacksrv/internal/handler"
)

func main() {
	// go turnOnProf()
	defer profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop()
	var addr, addrCtrl string
	flag.StringVar(&addr, "service", ":8080", "service address endpoint")
	flag.StringVar(&addrCtrl, "control", ":8081", "service address endpoint")
	flag.Parse()
	stopCh := make(chan interface{}, 5)
	stoppedCh := make(chan interface{})
	restartCh := make(chan interface{})
	lgr := log.New("server")
	lgr.SetHeader(`"level":"${level}","time":"${time_rfc3339_nano}","name":"${prefix}","location":"${short_file}:${line}"}`)
	lgr.SetLevel(getLogLVL())
	var srv *Server
	srv = NewServer(addr, lgr)

	go startControl(addrCtrl, restartCh)
	go srv.start(stopCh, stoppedCh)

	for {
		select {
		case <-restartCh:
			srv.lgr.Info("server restart signal received")
			stopCh <- struct{}{}
			srv.lgr.Info("issued stop signal for the server")
			srv.lstnr.Close()
			select {
			case <-stoppedCh:
				srv.lgr.Info("signal server stopped received, ready to restart .. ")
				srv = NewServer(addr, lgr)
				srv.lgr.Debug("new srv instance created")
				go srv.start(stopCh, stoppedCh)
			}
		}
	}
}

//Start with start you may either start or restart the server
func (srv *Server) start(stopCh <-chan interface{}, stoppedCh chan<- interface{}) {
	readingQueue := make(chan *conn.Conn, conn.MaxConn)
	stopWorkersCh := make(chan interface{})
	pool := conn.NewConnPool(srv.lgr, stopWorkersCh)
	tcpHandler := handler.NewTCP(srv.lgr, pool)
	srv.lgr.Infof("server started on address %s", srv.laddr)
	go tcpHandler.ConnListener(readingQueue, stopWorkersCh)
	for {
		select {
		case <-stopCh:
			srv.lstnr.Close()
			srv.lgr.Info("server stop signal received")
			close(stopWorkersCh)
			readingQueue = nil
			srv.lgr.Info("closing listeners.. ")
			//let every worker get its signals
			time.Sleep(1 * time.Second)
			stoppedCh <- struct{}{}
			srv.lgr.Info("server is stopped")
			return
		default:
		}
		conn, err := srv.lstnr.AcceptTCP()
		if err != nil {
			srv.lgr.Errorf("failed to accept conn: %", err)
			if conn != nil {
				conn.Close()
			}
			continue
		}
		srv.lgr.Infof("accepted tcp from ", conn.RemoteAddr())
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
func NewServer(addr string, lgr *log.Logger) *Server {
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
		lgr:     lgr,
		laddr:   addr,
	}
}

func startControl(addrCtrl string, restartCh chan<- interface{}) {
	laddr, err := net.ResolveTCPAddr("tcp", addrCtrl)
	l, err := net.ListenTCP("tcp", laddr)
	lgr := log.New("control")
	lgr.SetHeader(`"level":"${level}","name":"${prefix}","location":"${short_file}:${line}"}`)
	lgr.SetLevel(getLogLVL())
	if err != nil {
		fmt.Println("launch error ", err.Error())
		os.Exit(1)
	}
	for {
		lgr.Infof("ready to accept control commands on addr %s", addrCtrl)
		conn, err := l.AcceptTCP()
		if err != nil {
			lgr.Errorf("failed to accept conn: %", err)
			conn.Close()
			continue
		}
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		conn.SetKeepAlive(false)
		data := make([]byte, 3)
		if _, err := conn.Read(data); err != nil {
			lgr.Errorf("unable to read conn %v", err)
		}
		conn.Close()
		lgr.Infof("accepted tcp from ", conn.RemoteAddr())
		if string(data) == "rel" {
			restartCh <- struct{}{}
		}
	}
}

type Server struct {
	lstnr   *net.TCPListener
	tcpAddr *net.TCPAddr
	lgr     *log.Logger
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
