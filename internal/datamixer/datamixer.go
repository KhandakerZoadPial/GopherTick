package datamixer

import "gophertick/internal/dataproducer"

func Mixer(providers []dataproducer.Provider) chan dataproducer.Price {
	channel := make(chan dataproducer.Price)

	for _, provider := range providers {
		priceChannel := provider.Produce()

		go func(channelStream chan dataproducer.Price) {
			for price := range channelStream {
				channel <- price
			}
		}(priceChannel)
	}

	return channel
}
