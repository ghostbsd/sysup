package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// Setup our CLI Flags
var checkflag bool
var startflag bool
var websocketflag bool
func init() {
	flag.BoolVar(&checkflag, "check", false, "Check system status")
	flag.BoolVar(&startflag, "start", false, "Start and upgrade to latest packages")
	flag.BoolVar(&websocketflag, "websocket", false, "Start websocket server for direct API access and events")
	flag.Parse()
}

type Envelope struct {
	Method string
}

type Check struct {
	Updates bool
}
type InfoMsg struct {
	Info string
}

func parsejsonmsg(message []byte) int {
	if ( ! json.Valid(message) ) {
		log.Println("ERROR: Invalid JSON in return")
		return 1
	}
//	log.Printf("client-recv: %s", message)
	var env Envelope
	if err := json.Unmarshal(message, &env); err != nil {
		log.Fatal(err)
	}
	switch env.Method {
	case "check":
		var s struct {
			Envelope
			Check
		}
		if err := json.Unmarshal(message, &s); err != nil {
			log.Fatal(err)
		}
		var haveupdates bool = s.Updates
		if ( haveupdates ) {
			fmt.Println("The following updates are available")
			os.Exit(10)
		} else {
			fmt.Println("No updates available")
			os.Exit(0)
		}
	case "info":
		var s struct {
			Envelope
			InfoMsg
		}
		if err := json.Unmarshal(message, &s); err != nil {
			log.Fatal(err)
		}
		var infomsg string = s.Info
		fmt.Println(infomsg)

	default:
		log.Fatalf("unknown message type: %q", env.Method)
	}
	return 0
}

func startcheck() {
	data := map[string]string{
		"method": "check",
	}
	msg, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		log.Fatal("Failed encoding JSON:", err)
	}
	//fmt.Println("JSON Message: ", string(msg))
	send_err := c.WriteMessage(websocket.TextMessage, msg)
	if send_err != nil {
		log.Fatal("Failed talking to WS backend:", send_err)
	}

	done := make(chan struct{})
	defer close(done)


	// Wait for messages back
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		// Do things with the message back
		parsejsonmsg(message)
	}
}


// Start the websocket server
func startws() {
        log.SetFlags(0)
        http.HandleFunc("/ws", readws)
        log.Fatal(http.ListenAndServe(*addr, nil))
}

// Start our client connection to the WS server
var (
        c   *websocket.Conn
)
func connectws() {
	time.Sleep(2 * time.Second);
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
//	log.Printf("connecting to %s", u.String())

	err := errors.New("")
	var connected bool = false
	for attempt := 0; attempt < 10; attempt++ {
		c, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		if err == nil {
			connected = true
			break
		}
		time.Sleep(5 * time.Millisecond);
	}
	if (!connected) {
		log.Fatal("Failed connecting to websocket server", err)
	}
}

// Called when we want to signal that its time to close the WS connection
func closews() {
	log.Println("Closing WS connection")
	defer c.Close()

	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		return
	}
}

func main() {
	if len(os.Args) == 1 {
		flag.Usage()
	}

	// Capture any sigint
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		os.Exit(1)
	}()

	if ( checkflag ) {
		go startws()
		connectws()
		startcheck()
		closews()
		os.Exit(0)
	}
	if ( websocketflag ) {
		startws()
	}
}
