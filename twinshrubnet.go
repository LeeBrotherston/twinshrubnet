package twinshrubnet

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"net"
)

type UserSuppliedType[T any] any

// TreeNode is a node in the binary search tree
type TreeNode[T any] struct {
	binZero *TreeNode[T]
	binOne  *TreeNode[T]
	Value   UserSuppliedType[T]
}

// TreeRoot is the root of the binary search tree
type TreeRoot[T any] struct {
	ipv4 *TreeNode[T]
	ipv6 *TreeNode[T]
}

// NewTree returns the root of a new twinshrubnet tree
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

func (t *TreeRoot[T]) GetFromIPStr(ipStr string) (UserSuppliedType[T], *net.IPNet, error) {
	var (
		ipaddr net.IP
		err    error
	)
	ipaddr = net.ParseIP(ipStr)
	if ipaddr == nil {
		log.Printf("could not parse IP address=[%s], attempting to parse as CIDR\n", ipaddr)
		ipaddr, _, err = net.ParseCIDR(ipStr)
		if err != nil {
			return nil, nil, fmt.Errorf("could not parse IP address=[%s] as IP or CIDR, err=[%s]", ipStr, err)
		}
	}
	return t.GetFromIP(ipaddr)
}

func (t *TreeRoot[T]) GetFromIP(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	v4addr := ipaddr.To4()
	if v4addr != nil {
		return t.getFromIPv4(v4addr)
	} else {
		return t.getFromIPv6(ipaddr)
	}
}

func (t *TreeRoot[T]) getFromIPv4(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	var (
		network net.IPNet
	)

	location := t.ipv4
	v4Uint32 := binary.BigEndian.Uint32(ipaddr)

	for i := uint32(1); i < 34; i++ {
		// Keep Searching
		thing := v4bit(v4Uint32, i)
		var next *TreeNode[T]
		if thing == 0 {
			next = location.binZero
		} else {
			next = location.binOne
		}

		if next == nil {
			if location.Value == nil {
				return nil, nil, nil
			} else {
				network.IP = ipaddr
				network.Mask = net.CIDRMask(int(i-1), 32)
				return location.Value, &network, nil
			}
		}
		location = next
	}
	return nil, nil, fmt.Errorf("no results for search")
}

func (t *TreeRoot[T]) getFromIPv6(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	var (
		network net.IPNet
	)

	location := t.ipv6
	v6 := big.NewInt(0)
	v6.SetBytes(ipaddr)

	for i := 1; i <= 128; i++ {
		thing := v6.Bit(128 - i)
		var next *TreeNode[T]
		if thing == 0 {
			next = location.binZero
		} else {
			next = location.binOne
		}

		if next == nil {
			if location.Value == nil {
				return nil, nil, nil
			} else {
				network.IP = ipaddr
				network.Mask = net.CIDRMask(int(i-1), 128)
				return location.Value, &network, nil
			}
		}
		location = next
	}
	return nil, nil, fmt.Errorf("no results for search")
}

// v4bit is a simple function to return the n'th bit of the v4 uint32
func v4bit(v4 uint32, n uint32) uint {
	return uint((v4 >> (32 - n)) & 0x01)
}
