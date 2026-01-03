package dataflash

// Unit identifier to human-readable name mapping
// Based on ArduPilot's log_Units in libraries/AP_Logger/LogStructure.h
var unitNames = map[rune]string{
	'-': "",             // no units e.g. Pi, or a string
	'?': "UNKNOWN",      // Units which haven't been worked out yet
	'A': "A",            // Ampere
	'a': "Ah",           // Ampere hours
	'd': "deg",          // degrees (angular), -180 to 180
	'b': "B",            // bytes
	'B': "B/s",          // bytes per second
	'k': "deg/s",        // degrees per second
	'D': "deglatitude",  // degrees of latitude
	'e': "deg/s/s",      // degrees per second per second
	'E': "rad/s",        // radians per second
	'G': "Gauss",        // Gauss (magnetic field)
	'h': "degheading",   // 0 to 359.9
	'i': "A.s",          // Ampere second
	'J': "W.s",          // Joule (Watt second)
	'l': "l",            // litres
	'L': "rad/s/s",      // radians per second per second
	'm': "m",            // metres
	'n': "m/s",          // metres per second
	'o': "m/s/s",        // metres per second per second
	'O': "degC",         // degrees Celsius
	'%': "%",            // percent
	'S': "satellites",   // number of satellites
	's': "s",            // seconds
	't': "N.m",          // Newton meters (torque)
	'q': "rpm",          // rounds per minute
	'r': "rad",          // radians
	'U': "deglongitude", // degrees of longitude
	'u': "ppm",          // pulses per minute
	'v': "V",            // Volt
	'P': "Pa",           // Pascal
	'w': "Ohm",          // Ohm
	'W': "Watt",         // Watt
	'X': "W.h",          // Watt hour
	'y': "l/s",          // litres per second
	'Y': "us",           // pulse width modulation in microseconds
	'z': "Hz",           // Hertz
	'#': "instance",     // Sensor instance number
}

// Multiplier identifier to scaling factor mapping
// Based on ArduPilot's log_Multipliers in libraries/AP_Logger/LogStructure.h
// Note: Any adjustment implied by format field (e.g. "centi" in centidegrees) is
var multipliers = map[rune]float64{
	'-': 0,    // no multiplier e.g. a string
	'?': 1,    // multipliers which haven't been worked out yet
	'2': 1e2,  // x100
	'1': 1e1,  // x10
	'0': 1e0,  // x1
	'A': 1e-1, // /10
	'B': 1e-2, // /100
	'C': 1e-3, // /1000
	'D': 1e-4, // /10000
	'E': 1e-5, // /100000
	'F': 1e-6, // /1000000
	'G': 1e-7, // /10000000
	'I': 1e-9, // /1000000000
	'!': 3.6,  // (ampere*second => milliampere*hour) and (km/h => m/s)
	'/': 3600, // (ampere*second => ampere*hour)
}

// getUnitName returns the human-readable unit name for a unit identifier
func getUnitName(unitChar rune) string {
	if name, ok := unitNames[unitChar]; ok {
		return name
	}
	return ""
}

// getMultiplier returns the scaling factor for a multiplier identifier
func getMultiplier(multChar rune) float64 {
	if mult, ok := multipliers[multChar]; ok {
		return mult
	}
	return 1.0 // Default to no scaling
}
