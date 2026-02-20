package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gwugsuh/internal/bq25895"
)

type MockBQ struct {
	Status *bq25895.BQStatus
	Err    error
}

func (m *MockBQ) GetStatus() (*bq25895.BQStatus, error) {
	return m.Status, m.Err
}

type MockMax struct {
	Vol float64
	Soc float64
	Err error
}

func (m *MockMax) GetStatus() (float64, float64, error) {
	return m.Vol, m.Soc, m.Err
}

func TestRootHandler(t *testing.T) {
	tests := []struct {
		name           string
		bq             *MockBQ
		max            *MockMax
		expectedState  string
		expectedLevel  int
		expectedVol    float64
		expectedCharge bool
	}{
		{
			name: "Charging with Max and BQ",
			bq: &MockBQ{
				Status: &bq25895.BQStatus{
					ChargeStatus:      "Charging",
					Input:             "Connected",
					BatteryPercentage: 0.5,
					BatteryVoltage:    3.8,
				},
			},
			max: &MockMax{
				Vol: 3.9,
				Soc: 55.0,
			},
			expectedState:  "Charging",
			expectedLevel:  55,
			expectedVol:    3.9,
			expectedCharge: true,
		},
		{
			name: "Full Charge",
			bq: &MockBQ{
				Status: &bq25895.BQStatus{
					ChargeStatus: "Charging done",
					Input:        "Connected",
				},
			},
			max: &MockMax{
				Vol: 4.2,
				Soc: 100.0,
			},
			expectedState:  "Full",
			expectedLevel:  100,
			expectedVol:    4.2,
			expectedCharge: false,
		},
		{
			name: "Discharging (Not Charging + Disconnected)",
			bq: &MockBQ{
				Status: &bq25895.BQStatus{
					ChargeStatus: "Not Charging",
					Input:        "Disconnected",
				},
			},
			max: &MockMax{
				Vol: 3.7,
				Soc: 40.0,
			},
			expectedState:  "Discharging",
			expectedLevel:  40,
			expectedVol:    3.7,
			expectedCharge: false,
		},
		{
			name: "Not Charging (Connected)",
			bq: &MockBQ{
				Status: &bq25895.BQStatus{
					ChargeStatus: "Not Charging",
					Input:        "Connected",
				},
			},
			max: &MockMax{
				Vol: 3.7,
				Soc: 40.0,
			},
			expectedState:  "Not Charging",
			expectedLevel:  40,
			expectedVol:    3.7,
			expectedCharge: false,
		},
		{
			name: "Fallback to BQ when Max fails",
			bq: &MockBQ{
				Status: &bq25895.BQStatus{
					ChargeStatus:      "Not Charging", // Raw status
					Input:             "Disconnected",
					BatteryPercentage: 0.3,
					BatteryVoltage:    3.6,
				},
			},
			max: &MockMax{
				Err: errors.New("max failure"), 
			},
			expectedState:  "Discharging",
			expectedLevel:  30, // 0.3 * 100
			expectedVol:    3.6,
			expectedCharge: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				bq:  tt.bq,
				max: tt.max,
			}
			
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()

			s.rootHandler(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			var br BatteryResponse
			if err := json.NewDecoder(resp.Body).Decode(&br); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if br.State != tt.expectedState {
				t.Errorf("Expected State %s, got %s", tt.expectedState, br.State)
			}
			if br.Level != tt.expectedLevel {
				t.Errorf("Expected Level %d, got %d", tt.expectedLevel, br.Level)
			}
			if br.Voltage != tt.expectedVol {
				t.Errorf("Expected Voltage %f, got %f", tt.expectedVol, br.Voltage)
			}
			if br.IsCharging != tt.expectedCharge {
				t.Errorf("Expected IsCharging %v, got %v", tt.expectedCharge, br.IsCharging)
			}
		})
	}
}
