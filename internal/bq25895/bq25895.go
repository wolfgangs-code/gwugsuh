package bq25895

import (
	"periph.io/x/conn/v3/i2c"
	"time"
)

const (
	Addr = 0x6A
	
	REG_ILIM     = 0x00
	REG_ICHG     = 0x04
	REG_WATCHDOG = 0x07
	REG_BATFET   = 0x09
	REG_STATUS   = 0x0B
	REG_BATV     = 0x0E
	REG_CONV_ADC = 0x02
	
	BYTE_WATCHDOG_STOP = 0b10001101
	BYTE_ILIM_2A       = 0b01101000
	BYTE_ILIM_3A       = 0b01111100
	BYTE_ILIM_3_25A    = 0b01111111
	BYTE_ICHG_0_5A     = 0b01111111 // Default in script
	BYTE_BATFET        = 0b01001000 // Delay before battery disconnected
	
	BYTE_CONV_ADC_START = 0b10011101
	BYTE_CONV_ADC_STOP  = 0b00011101
	
	REG_BATFET_DIS  = 0x09
	BYTE_BATFET_DIS = 0b01101000
)

type BQ25895 struct {
	dev *i2c.Dev
}

type BQStatus struct {
	Input             string
	ChargeStatus      string
	BatteryVoltage    float64
	BatteryPercentage float64
	TimeRemaining     int
}

func NewBQ25895(bus i2c.Bus) (*BQ25895, error) {
	dev := &i2c.Dev{Addr: Addr, Bus: bus}
	return &BQ25895{dev: dev}, nil
}

func (b *BQ25895) Init(inputLimit string) error {
	// Reset Watchdog
	if err := b.writeReg(REG_WATCHDOG, BYTE_WATCHDOG_STOP); err != nil {
		return err
	}
	
	// Set Input Limit
	ilim := BYTE_ILIM_3_25A
	switch inputLimit {
	case "2A":
		ilim = BYTE_ILIM_2A
	case "3A":
		ilim = BYTE_ILIM_3A
	}
	if err := b.writeReg(REG_ILIM, byte(ilim)); err != nil {
		return err
	}

	// Set Charge Current
	if err := b.writeReg(REG_ICHG, BYTE_ICHG_0_5A); err != nil {
		return err
	}
	
	// Set BATFET
	if err := b.writeReg(REG_BATFET, BYTE_BATFET); err != nil {
		return err
	}
	
	return nil
}

func (b *BQ25895) GetStatus() (*BQStatus, error) {
	// Start ADC Conversion
	if err := b.writeReg(REG_CONV_ADC, BYTE_CONV_ADC_START); err != nil {
		return nil, err
	}
	
	// Wait for conversion (Script says 1.2s? That seems long but okay)
	time.Sleep(1200 * time.Millisecond)
	
	// Read Status
	statusByte, err := b.readReg(REG_STATUS)
	if err != nil {
		return nil, err
	}
	
	// Read BatV
	batVByte, err := b.readReg(REG_BATV)
	if err != nil {
		return nil, err
	}
	
	// Stop ADC Conversion
	if err := b.writeReg(REG_CONV_ADC, BYTE_CONV_ADC_STOP); err != nil {
		return nil, err
	}
	
	// Parse Status
	// Bit 2 (0-indexed) is PG_STAT (Power Good)
	pgStat := (statusByte >> 2) & 1
	power := "Disconnected"
	if pgStat == 1 {
		power = "Connected"
	}
	
	// Bits 4,3 are CHRG_STAT
	chrgStat := (statusByte >> 3) & 0b11
	charge := "Discharging"
	switch chrgStat {
	case 0b00:
		charge = "Not Charging"
	case 0b01:
		charge = "Pre-Charge"
	case 0b10:
		charge = "Charging"
	case 0b11:
		charge = "Charging done"
	}
	
	// Parse Voltage
	// Bit 7 is VBUS_STAT (ignored in calculation here, using BATV reg)
	// REG_BATV bits:
	// Bit 6: 1.280V
	// Bit 5: 0.640V
	// ...
	// Offset: 2.304V
	
	batV := 2.304
	if (batVByte>>6)&1 == 1 { batV += 1.280 }
	if (batVByte>>5)&1 == 1 { batV += 0.640 }
	if (batVByte>>4)&1 == 1 { batV += 0.320 }
	if (batVByte>>3)&1 == 1 { batV += 0.160 }
	if (batVByte>>2)&1 == 1 { batV += 0.080 }
	if (batVByte>>1)&1 == 1 { batV += 0.040 }
	if (batVByte>>0)&1 == 1 { batV += 0.020 }
	
	// Translate Voltage to Percentage
	// Range 3.5V to 4.184V
	batPercent := bqTranslate(batV, 3.5, 4.184, 0, 1)
	if batPercent < 0 { batPercent = 0 }
	if batPercent > 1 { batPercent = 1 }
	
	// Time Remaining
	// Script: int( batpercent * 60* BAT_CAPACITY / CURRENT_DRAW)
	// BAT_CAPACITY = 2500, CURRENT_DRAW = 2000
	const batCapacity = 2500.0
	const currentDraw = 2000.0
	timeLeftMin := int(batPercent * 60 * batCapacity / currentDraw)
	
	if power == "Connected" {
		timeLeftMin = -1
	}
	
	// Low Voltage protection logic from script
	if batV < 3.5 {
		// Disable BATFET
		b.writeReg(REG_BATFET_DIS, BYTE_BATFET_DIS)
	}

	return &BQStatus{
		Input:             power,
		ChargeStatus:      charge,
		BatteryVoltage:    batV,
		BatteryPercentage: batPercent,
		TimeRemaining:     timeLeftMin,
	}, nil
}

func (b *BQ25895) writeReg(reg byte, val byte) error {
	return b.dev.Tx([]byte{reg, val}, nil)
}

func (b *BQ25895) readReg(reg byte) (byte, error) {
	buf := make([]byte, 1)
	if err := b.dev.Tx([]byte{reg}, buf); err != nil {
		return 0, err
	}
	return buf[0], nil
}

func bqTranslate(val, inFrom, inTo, outFrom, outTo float64) float64 {
	outRange := outTo - outFrom
	inRange := inTo - inFrom
	inVal := val - inFrom
	valNorm := (inVal / inRange) * outRange
	return outFrom + valNorm
}
