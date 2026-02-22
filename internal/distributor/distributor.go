package distributor

import (
	"gophertick/internal/dataproducer"
)

type Hub struct {
	clients map[chan dataproducer.Price]bool

	Register   chan chan dataproducer.Price
	Unregister chan chan dataproducer.Price
	Broadcast  chan dataproducer.Price
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[chan dataproducer.Price]bool),
		Register:   make(chan chan dataproducer.Price),
		Unregister: make(chan chan dataproducer.Price),
		Broadcast:  make(chan dataproducer.Price),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.clients[client] = true

		case client := <-h.Unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client)
			}
		case message := <-h.Broadcast:
			for client := range h.clients {

				select {
				case client <- message:
				default:
				}
			}

		}

	}
}
