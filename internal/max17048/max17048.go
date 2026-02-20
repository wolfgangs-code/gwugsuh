package max17048

import (
	"periph.io/x/conn/v3/i2c"
)

const Addr = 0x36

type MAX17048 struct {
	dev *i2c.Dev
}

func NewMAX17048(bus i2c.Bus) (*MAX17048, error) {
	dev := &i2c.Dev{Addr: Addr, Bus: bus}
	// Init: 0xFE, 0xFFFF (Reset command from original script)
	// The original script does: bus.write_word_data(MAX17048_ADDR, 0xFE ,0xFFFF)
	// This seems to reset the gauge.
	// Note: write_word_data writes LSByte first. 0xFFFF is symmetric.
	return &MAX17048{dev: dev}, nil
}

func (m *MAX17048) Init() error {
	// 0xFE is command register for reset?
	// Original: bus.write_word_data(MAX17048_ADDR, 0xFE ,0xFFFF)
	_, err := m.dev.Write([]byte{0xFE, 0xFF, 0xFF})
	return err
}

func (m *MAX17048) GetStatus() (voltage float64, soc float64, err error) {
	// Voltage: Reg 0x02
	// SOC: Reg 0x04
	
	buf := make([]byte, 2)
	
	// Read Voltage
	if err := m.dev.Tx([]byte{0x02}, buf); err != nil {
		return 0, 0, err
	}
	// Original: (((max17048_v_16 & 0x00FF) << 8) + (max17048_v_16 >> 8))
	// SMBus read_word_data reads low byte first. 
	// So buf[0] is LSB, buf[1] is MSB.
	// The Python code swaps bytes? 
	// (max17048_v_16 & 0x00FF) << 8  -> Takes LSB and moves to MSB
	// (max17048_v_16 >> 8) -> Takes MSB and moves to LSB
	// So it effectively interprets the word as Big Endian (MSB first)? 
	// Or it swaps because the chip sends MSB first but SMBus reads LSB first?
	// MAX17048 datasheet: "Register data is read and written LSB first"
	// Actually typical I2C is MSB first.
	// Let's trust the python math:
	// If read_word_data returns 0x1234 (where 0x34 is at address, 0x12 at address+1)
	// Python: (0x34 << 8) + 0x12 = 0x3412.
	// So it swaps the bytes.
	
	rawV := uint16(buf[0])<<8 | uint16(buf[1]) // Construct swapped
	voltage = float64(rawV) * 78.125 / 1000000.0

	// Read SOC
	if err := m.dev.Tx([]byte{0x04}, buf); err != nil {
		return 0, 0, err
	}
	rawSOC := uint16(buf[0])<<8 | uint16(buf[1])
	soc = float64(rawSOC) / 256.0

	return voltage, soc, nil
}
