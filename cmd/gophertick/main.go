package main

import (
	"gophertick/internal/datamixer"
	"gophertick/internal/dataproducer"
	"gophertick/internal/distributor"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  256,
	WriteBufferSize: 256,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func serveWS(h *distributor.Hub, w http.ResponseWriter, r *http.Request) {
	connection, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("Upgrade error:", err)
	}
	defer connection.Close()

	clientChannel := make(chan dataproducer.Price)
	h.Register <- clientChannel

	defer func() {
		h.Unregister <- clientChannel
	}()

	for price := range clientChannel {
		connection.WriteJSON(price)
	}

}

func main() {
	bitCoin := dataproducer.Provider{Name: "Bitcoin"}
	etherium := dataproducer.Provider{Name: "Etherium"}
	binance := dataproducer.Provider{Name: "Binance"}

	allProviders := []dataproducer.Provider{}
	allProviders = append(allProviders, bitCoin)
	allProviders = append(allProviders, etherium)
	allProviders = append(allProviders, binance)

	realTimePrice := datamixer.Mixer(allProviders)

	hub := distributor.NewHub()
	go hub.Run()

	go func() {
		for price := range realTimePrice {
			hub.Broadcast <- price
		}
	}()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	})

	http.Handle("/", http.FileServer(http.Dir("../../ui")))

	log.Println("GopherTick Aggregator live on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
