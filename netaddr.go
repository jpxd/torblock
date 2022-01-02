package torblock

import "fmt"

// IPv4 is a comparable representation of a 32bit IPv4 address
type IPv4 struct {
	addr uint32
}

// CreateIPv4 returns the IPv4 of the address a.b.c.d.
func CreateIPv4(a, b, c, d uint8) IPv4 {
	return IPv4{
		addr: uint32(uint32(a)<<24 | uint32(b)<<16 | uint32(c)<<8 | uint32(d)),
	}
}

// ParseIPv4 parses s as an IPv4 address, returning the result or an error (adapted from inet.af/netaddr)
func ParseIPv4(s string) (IPv4, error) {
	var fields [3]uint8
	var val, pos int
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			val = val*10 + int(s[i]) - '0'
			if val > 255 {

				return IPv4{}, fmt.Errorf("field has value >255")
			}
		} else if s[i] == '.' {
			if i == 0 || i == len(s)-1 || s[i-1] == '.' {
				return IPv4{}, fmt.Errorf("every field must have at least one digit")
			}
			if pos == 3 {
				return IPv4{}, fmt.Errorf("address too long")
			}
			fields[pos] = uint8(val)
			pos++
			val = 0
		} else {
			return IPv4{}, fmt.Errorf("unexpected character")
		}
	}
	if pos < 3 {
		return IPv4{}, fmt.Errorf("address too short")
	}
	return CreateIPv4(fields[0], fields[1], fields[2], uint8(val)), nil
}

// IPv4Set contains a set of IPv4 addresses
type IPv4Set struct {
	set map[IPv4]bool
}

// CreateIPv4Set creates a new empty IPv4Set
func CreateIPv4Set() *IPv4Set {
	return &IPv4Set{
		map[IPv4]bool{},
	}
}

// Add appends a new IPv4 to the set
func (s *IPv4Set) Add(ip IPv4) {
	s.set[ip] = true
}

// Contains checks for an existing IPv4 within the set
func (s *IPv4Set) Contains(ip IPv4) bool {
	return s.set[ip]
}
