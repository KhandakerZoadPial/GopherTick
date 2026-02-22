package dataproducer

import (
	"math/rand/v2"
	"time"
)

type Provider struct {
	Name string
}

type Price struct {
	ProviderName string    `json:"provider_name"`
	CurrentPrice float64   `json:"current_price"`
	Timestamp    time.Time `json:"timestamp"`
}

type DataProducer interface {
	Produce() chan Price
}

func (p *Provider) Run(ch chan Price) {

	for {
		time.Sleep(100 * time.Millisecond)
		ch <- Price{
			ProviderName: p.Name,
			CurrentPrice: 100 + rand.Float64()*10,
			Timestamp:    time.Now(),
		}
	}

}

func (p *Provider) Produce() chan Price {
	channel := make(chan Price)
	go p.Run(channel)

	return channel

}
