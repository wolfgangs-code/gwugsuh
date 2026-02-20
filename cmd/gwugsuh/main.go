package main

import (
	"log"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"

	"gwugsuh/internal/bq25895"
	"gwugsuh/internal/max17048"
	"gwugsuh/internal/server"
)

func main() {
	log.Println("Starting gwugsuh...")

	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatalf("failed to open I2C: %v", err)
	}
	defer bus.Close()

	bq, err := bq25895.NewBQ25895(bus)
	if err != nil {
		log.Printf("Failed to init BQ25895: %v", err)
	} else {
		// Initialize the chip with default 3.25A input limit
		if err := bq.Init("3.25A"); err != nil {
			log.Printf("Failed to configure BQ25895: %v", err)
		}
	}

	mx, err := max17048.NewMAX17048(bus)
	if err != nil {
		log.Printf("Failed to init MAX17048: %v", err)
	} else {
		if err := mx.Init(); err != nil {
			log.Printf("Failed to configure MAX17048: %v", err)
		}
	}

	log.Printf("Hardware Initialized: BQ25895 (Addr: 0x%X), MAX17048 (Addr: 0x%X)", bq25895.Addr, max17048.Addr)

	if err := server.Run(3000, bq, mx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
