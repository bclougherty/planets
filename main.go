package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/octoberxp/planets/controllers"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/octoberxp/glaze"
	"github.com/spf13/viper"
)

var port int
var graceful bool
var verbose bool

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.IntVar(&port, "port", 80, "What port to listen on")
	flag.BoolVar(&verbose, "verbose", false, "log every transaction")
	flag.BoolVar(&graceful, "graceful", false, "the already-open fd to listen on (internal use only)")

	log.SetPrefix(fmt.Sprintf("[%5d] ", syscall.Getpid()))
}

type counter struct {
	m sync.Mutex
	c int
}

func (c *counter) get() (ct int) {
	c.m.Lock()
	ct = c.c
	c.m.Unlock()
	return
}

var connCount counter

type gracefulConn struct {
	net.Conn
}

func (w gracefulConn) Close() error {
	if verbose {
		log.Printf("close on conn to %v", w.RemoteAddr())
	}

	connCount.m.Lock()
	connCount.c--
	connCount.m.Unlock()

	return w.Conn.Close()
}

type signal struct{}

type gracefulListener struct {
	net.Listener
	stop    chan signal
	stopped bool
}

var netListener gracefulListener

func newGracefulListener(l net.Listener) (sl gracefulListener) {
	sl = gracefulListener{Listener: l, stop: make(chan signal, 1)}

	// this goroutine monitors the channel. Can't do this in
	// Accept (below) because once it enters sl.Listener.Accept()
	// it blocks. We unblock it by closing the fd it is trying to
	// accept(2) on.
	go func() {
		_ = <-sl.stop
		sl.stopped = true
		sl.Listener.Close()
	}()
	return
}

func (gl gracefulListener) File() *os.File {
	tl := gl.Listener.(*net.TCPListener)
	fl, _ := tl.File()
	return fl
}

func (gl gracefulListener) Accept() (c net.Conn, err error) {
	c, err = gl.Listener.Accept()
	if err != nil {
		return
	}

	// Wrap the returned connection, so that we can observe when
	// it is closed.
	c = gracefulConn{Conn: c}

	// Count it
	connCount.m.Lock()
	connCount.c++
	connCount.m.Unlock()

	return
}

func logreq(req *http.Request) {
	if verbose {
		log.Printf("%v %v from %v", req.Method, req.URL, req.RemoteAddr)
	}
}

func parseConfig() error {
	// Create configuration
	viper.Set("AppRoot", os.ExpandEnv("."))

	viper.SetConfigName("config")
	viper.AddConfigPath(os.ExpandEnv("./config/"))
	err := viper.ReadInConfig()

	if err != nil {
		return err
	}

	viper.Set("FullViewPath", path.Join(viper.GetString("AppRoot"), viper.GetString("ViewPath")))

	return nil
}

func upgradeServer(w http.ResponseWriter, req *http.Request) {
	file := netListener.File() // this returns a Dup()

	cmd := exec.Command("./planets",
		fmt.Sprintf("-port=%d", port),
		fmt.Sprintf("-verbose=%v", verbose),
		"-graceful")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{file}

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Graceful restart: Failed to launch, error: %v", err)
	}
}

func main() {
	flag.Parse()

	rand.Seed(time.Now().UTC().UnixNano())

	err := parseConfig()
	if nil != err {
		log.Fatalf("Error reading in configuration: %s", err)
	}

	funcMap := template.FuncMap{
		"safeHtml": glaze.SafeHTML,
	}

	// Instantiate controllers
	public := controllers.NewPublicController(funcMap)

	// Create routes
	router := mux.NewRouter()

	for path, handler := range glaze.GenerateRoutes(public) {
		if path == "/public/index" {
			router.HandleFunc("/", handler)
		} else {
			router.HandleFunc(path, handler)
		}
	}

	accessLogFilename := "logs/access.log"
	var accessLog *os.File

	if _, err := os.Stat(accessLogFilename); os.IsNotExist(err) {
		_, err := os.Create(accessLogFilename)
		if err != nil {
			log.Fatalf("Failed to create log file at %s", accessLogFilename)
		}
	}

	accessLog, err = os.OpenFile(accessLogFilename, os.O_APPEND|os.O_WRONLY, 0600)

	if err != nil {
		panic(err)
	}

	mainLogFilename := "logs/planets.log"
	var mainLog *os.File

	if _, err := os.Stat(mainLogFilename); os.IsNotExist(err) {
		_, err := os.Create(mainLogFilename)
		if err != nil {
			log.Fatalf("Failed to create log file at %s", mainLogFilename)
		}
	}

	mainLog, err = os.OpenFile(mainLogFilename, os.O_APPEND|os.O_WRONLY, 0600)

	if err != nil {
		panic(err)
	}

	log.SetOutput(mainLog)

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("views/static"))))

	loggingRouter := handlers.LoggingHandler(accessLog, router)

	http.HandleFunc("/upgrade", upgradeServer)

	http.Handle("/", loggingRouter)

	log.Printf("Attempting to create server as :%d", port)

	// Set up listener
	var l net.Listener
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}

	if graceful {
		log.Print("Listening to existing file descriptor")
		f := os.NewFile(3, "")
		l, err = net.FileListener(f)
	} else {
		log.Print("Listening on a new file descriptor")
		l, err = net.Listen("tcp", server.Addr)
	}
	if err != nil {
		log.Fatal(err)
	}

	netListener = newGracefulListener(l)

	if graceful {
		parent := syscall.Getppid()
		log.Printf("main: Killing parent pid: %v", parent)
		syscall.Kill(parent, syscall.SIGTERM)
	}

	log.Printf("Serving on http://localhost:%d/", port)
	err = server.Serve(netListener)

	if err != nil {
		log.Fatal(err)
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	http.Error(w, "Could not find the requested page", http.StatusInternalServerError)
}
