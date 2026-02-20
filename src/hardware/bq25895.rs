// BQ25895 by Texas Instruments
// An I2C battery charger chip

const BQ25895_ADDR: u16 = 0x6A;

// Registers
const REG_ILIM: u8 = 0x00; // Input Current Limit
const REG_ICHG: u8 = 0x04; // Fast Charge Current
const REG_WATCHDOG: u8 = 0x07; // Watchdog Timer Control
const REG_BATFET: u8 = 0x09; // Battery FET Control
const REG_STATUS: u8 = 0x0B; // System Status Register
const REG_CONV_ADC: u8 = 0x02; // ADC Control
const REG_BATV: u8 = 0x0E; // Battery Voltage ADC Result

// Configuration Values
const BYTE_WATCHDOG_STOP: u8 = 0b1000_1101; // Stop Watchdog timer
const BYTE_ILIM: u8 = 0b0111_1111; // 3.25A input current limit
const BYTE_ICHG: u8 = 0b0111_1111; // .5A charging current limit
const BYTE_BATFET: u8 = 0b0100_1000; // Delay before battery == onnected
const BYTE_BATFET_DIS: u8 = 0b0110_1000;
const BYTE_CONV_ADC_START: u8 = 0b1001_1101;
const BYTE_CONV_ADC_STOP: u8 = 0b0001_11
const BAT_CAPACITY: u16 = 2500; // Battery capacity in mAh
const CURRENT_DRAW: u16 = 2000; // Current draw in mAh