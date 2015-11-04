package client

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/HackerLoop/rotonde/shared"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
)

func startConnection(rotondeUrl string, inChan, outChan chan interface{}) {
	log.Println("startRotondeClient")
	u, err := url.Parse(rotondeUrl)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := net.Dial("tcp", u.Host)
		if err != nil {
			log.Println(err)
			time.Sleep(2 * time.Second)
			continue
		}
		ws, response, err := websocket.NewClient(conn, u, http.Header{}, 10000, 10000)
		if err != nil {
			log.Println(err)
			log.Println(response)
			time.Sleep(2 * time.Second)
			continue
		}
		processRotondePackets(ws, inChan, outChan)
	}
}

func processRotondePackets(conn *websocket.Conn, inChan, outChan chan interface{}) {
	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		var jsonPacket []byte
		var err error
		var packet rotonde.Packet

		for {
			select {
			case dispatcherPacket := <-inChan:
				switch data := dispatcherPacket.(type) {
				case rotonde.Event:
					packet = rotonde.Packet{Type: "event", Payload: data}
				case rotonde.Action:
					packet = rotonde.Packet{Type: "action", Payload: data}
				case rotonde.Subscription:
					packet = rotonde.Packet{Type: "sub", Payload: data}
				case rotonde.Unsubscription:
					packet = rotonde.Packet{Type: "unsub", Payload: data}
				case rotonde.Definition:
					packet = rotonde.Packet{Type: "def", Payload: data}
				case rotonde.UnDefinition:
					packet = rotonde.Packet{Type: "undef", Payload: data}
				default:
					log.Info("Oops unknown packet: ", dispatcherPacket)
				}

				jsonPacket, err = json.Marshal(packet)
				if err != nil {
					log.Warning(err)
				}

				if err := conn.WriteMessage(websocket.TextMessage, jsonPacket); err != nil {
					log.Warning(err)
					return
				}

			case <-errChan:
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var dispatcherPacket interface{}

		for {
			messageType, reader, err := conn.NextReader()
			if err != nil {
				log.Println(err)
				errChan <- err
				return
			}
			if messageType == websocket.TextMessage {
				packet := rotonde.Packet{}
				decoder := json.NewDecoder(reader)
				if err := decoder.Decode(&packet); err != nil {
					log.Warning(err)
					continue
				}

				switch packet.Type {
				case "event":
					event := rotonde.Event{}
					mapstructure.Decode(packet.Payload, &event)
					dispatcherPacket = event
				case "action":
					action := rotonde.Action{}
					mapstructure.Decode(packet.Payload, &action)
					dispatcherPacket = action
				case "def":
					definition := rotonde.Definition{}
					mapstructure.Decode(packet.Payload, &definition)
					dispatcherPacket = definition
				case "undef":
					unDefinition := rotonde.UnDefinition{}
					mapstructure.Decode(packet.Payload, &unDefinition)
					dispatcherPacket = unDefinition
				}

				outChan <- dispatcherPacket
			}
		}
	}()

	log.Println("Treating messages")
	wg.Wait()
}
