package twinshrubnet

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"net"
)

type UserSuppliedType[T any] any

// Binary tree struct
type TreeNode[T any] struct {
	binZero *TreeNode[T]
	binOne  *TreeNode[T]
	Value   UserSuppliedType[T]
}

type TreeRoot[T any] struct {
	ipv4 *TreeNode[T]
	ipv6 *TreeNode[T]
}

func NewTree[T any]() *TreeRoot[T] {
	return &TreeRoot[T]{
		ipv4: &TreeNode[T]{},
		ipv6: &TreeNode[T]{},
	}
}

// AddNet add's a network to the tree, returning a pointer to the node representing that network (or error)
func (t *TreeRoot[T]) AddNet(cidr string, userdata T) (*TreeNode[T], error) {
	var location *TreeNode[T]
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	// IP address already masked off
	maskOnes, bitsize := ipnet.Mask.Size()

	if bitsize == 32 {
		// IPv4
		location = t.ipv4
		v4Uint32 := binary.BigEndian.Uint32(ipnet.IP)

		//bitmask := uint32(1)
		for i := uint32(1); i <= uint32(maskOnes); i++ {
			thing := v4bit(v4Uint32, i)
			if thing == 0 {
				if location.binZero == nil {
					location.binZero = &TreeNode[T]{}
				}
				location = location.binZero
			} else {
				if location.binOne == nil {
					location.binOne = &TreeNode[T]{}
				}
				location = location.binOne
			}
		}

		location.Value = userdata
		return location, nil

	} else if bitsize == 128 {
		// IPv6
		location = t.ipv6

		v6 := big.NewInt(0)
		v6.SetBytes(ipnet.IP)
		for i := 1; i <= maskOnes; i++ {
			thing := v6.Bit(128 - i)
			if thing == 0 {
				if location.binZero == nil {
					location.binZero = &TreeNode[T]{}
				}
				location = location.binZero
			} else {
				if location.binOne == nil {
					location.binOne = &TreeNode[T]{}
				}
				location = location.binOne
			}
		}
	}

	location.Value = userdata
	return location, nil
}

func (t *TreeRoot[T]) FindNetFromIP(ipStr string) (UserSuppliedType[T], error) {
	var (
		ipaddr net.IP
		subnet *net.IPNet
		err    error
	)
	ipaddr = net.ParseIP(ipStr)
	if ipaddr == nil {
		log.Printf("could not parse IP address=[%s], attempting to parse as CIDR\n", ipaddr)
		ipaddr, subnet, err = net.ParseCIDR(ipStr)
		if err != nil {
			return nil, fmt.Errorf("could not parse IP address=[%s] as IP or CIDR, err=[%s]", ipStr, err)
		} else {
			ones, bits := subnet.Mask.Size()
			if ones == bits {
				log.Printf("CIDR parsed as single host subnet, using IP=[%s]\n", ipaddr.String())
			} else {
				log.Printf("Provided IP=[%s] parses as a CIDR block, attempting to use [%s] as IP\n", ipStr, ipaddr.String())
			}
		}
	}

	v4addr := ipaddr.To4()
	if v4addr != nil {
		// IPv4
		location := t.ipv4
		v4Uint32 := binary.BigEndian.Uint32(v4addr)

		//bitmask := uint32(1)
		for i := uint32(1); ; i++ {

			// Keep Searching
			thing := v4bit(v4Uint32, i)
			if thing == 0 {
				if location.binZero == nil {
					if location.Value == nil {
						return nil, nil
					} else {
						return location.Value, nil
					}
				}
				location = location.binZero
			} else {
				if location.binOne == nil {
					if location.Value == nil {
						return nil, nil
					} else {
						return location.Value, nil
					}
				}
				location = location.binOne
			}
		}
	} else {
		// IPv6
		location := t.ipv6
		v6 := big.NewInt(0)
		v6.SetBytes(ipaddr)
		for i := 1; i <= 128; i++ {
			thing := v6.Bit(128 - i)
			if thing == 0 {
				if location.binZero == nil {
					if location.Value == nil {
						return nil, nil
					} else {
						return location.Value, nil
					}
				}
				location = location.binZero
			} else {
				if location.binOne == nil {
					if location.Value == nil {
						return nil, nil
					} else {
						return location.Value, nil
					}
				}
				location = location.binOne
			}
		}
	}
	return nil, fmt.Errorf("no results for search")
}

func v4bit(v4 uint32, bitloc uint32) uint {
	return uint((v4 >> (32 - bitloc)) & 0x01)
}
