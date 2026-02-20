package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"gwugsuh/internal/bq25895"
)

type BQClient interface {
	GetStatus() (*bq25895.BQStatus, error)
}

type MaxClient interface {
	GetStatus() (float64, float64, error)
}

type BatteryResponse struct {
	Level      int     `json:"sensor.battery_level"`
	Voltage    float64 `json:"sensor.battery_voltage"`
	State      string  `json:"sensor.battery_state"`
	IsCharging bool    `json:"sensor.is_charging"`
}

type Server struct {
	bq  BQClient
	max MaxClient
}

func Run(port int, bq BQClient, max MaxClient) error {
	s := &Server{
		bq:  bq,
		max: max,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.rootHandler)

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Listening on %s", addr)
	return srv.ListenAndServe()
}

func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	// Defaults
	resp := BatteryResponse{
		State: "Discharging", // Default assumption if we can't read anything
	}

	// Fetch BQ25895 status
	var bqStatus *bq25895.BQStatus
	if s.bq != nil {
		var err error
		bqStatus, err = s.bq.GetStatus()
		if err != nil {
			log.Printf("Error reading BQ25895: %v", err)
		}
	}

	// Fetch MAX17048 status
	var maxVol, maxSoc float64
	var maxErr error
	if s.max != nil {
		maxVol, maxSoc, maxErr = s.max.GetStatus()
		if maxErr != nil {
			log.Printf("Error reading MAX17048: %v", maxErr)
		}
	}

	// Logic for Level
	if s.max != nil && maxErr == nil {
		resp.Level = int(maxSoc)
	} else if bqStatus != nil {
		resp.Level = int(bqStatus.BatteryPercentage * 100)
	}

	// Logic for Voltage
	if s.max != nil && maxErr == nil {
		resp.Voltage = maxVol
	} else if bqStatus != nil {
		resp.Voltage = bqStatus.BatteryVoltage
	}

	// Logic for State & IsCharging
	if bqStatus != nil {
		switch bqStatus.ChargeStatus {
		case "Charging done":
			resp.State = "Full"
		case "Charging", "Pre-Charge":
			resp.State = "Charging"
		case "Not Charging":
			if bqStatus.Input == "Connected" {
				resp.State = "Not Charging"
			} else {
				resp.State = "Discharging"
			}
		default:
			resp.State = "Discharging"
		}
	}

	resp.IsCharging = (resp.State == "Charging")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
