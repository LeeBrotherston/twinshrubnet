package twinshrubnet

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"net"
	"sync"
)

type UserSuppliedType[T any] any

// TreeNode is a node in the binary search tree
type TreeNode[T any] struct {
	binZero  *TreeNode[T]
	binOne   *TreeNode[T]
	valuePtr *UserSuppliedType[T]
}

// TreeRoot is the root of the binary search tree
type TreeRoot[T any] struct {
	ipv4 *TreeNode[T]
	ipv6 *TreeNode[T]
	lock *sync.RWMutex
}

// NewTree returns the root of a new twinshrubnet tree
func NewTree[T any]() *TreeRoot[T] {
	return &TreeRoot[T]{
		ipv4: &TreeNode[T]{},
		ipv6: &TreeNode[T]{},
		lock: new(sync.RWMutex),
	}
}

// AddNet add's a network to the tree, returning a pointer to the node representing that network (or error)
func (t *TreeRoot[T]) AddNet(cidr string, userdata T) (*TreeNode[T], error) {
	t.lock.Lock()
	defer t.lock.Unlock()

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

		location.valuePtr = new(UserSuppliedType[T])
		*location.valuePtr = userdata

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

	location.valuePtr = new(UserSuppliedType[T])
	*location.valuePtr = userdata

	return location, nil
}

// RemoveNet removes a network from the tree...  Currently it is not complete,
// only removing the value rather than the tree entries themselves, but we will
// add that sortly.  GO's garbage collection should handle freeing up the memory
// used by the value
func (t *TreeRoot[T]) RemoveNet(cidr string) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	location, err := t.findNodeFromIPNet(*ipnet)
	if err != nil {
		return err
	}

	location.valuePtr = nil
	return nil
}

func (t *TreeRoot[T]) findNodeFromIPNet(network net.IPNet) (*TreeNode[T], error) {
	var node *TreeNode[T]
	switch ipVersion(&network) {
	case 4:
		node, _ = t.getNodeFromIPv4(network.IP)
	case 6:
		node, _ = t.getNodeFromIPv6(network.IP)
	default:
		return nil, fmt.Errorf("could not determine IP type")
	}
	return node, nil
}

// getFromIPv4 find the appropriate node given an address
func (t *TreeRoot[T]) getNodeFromIPv4(ipaddr net.IP) (*TreeNode[T], uint32) {
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
			// Found it
			return location, (i - 1)
		}
		location = next
	}
	return nil, 0
}

func (t *TreeRoot[T]) getNodeFromIPv6(ipaddr net.IP) (*TreeNode[T], int) {
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
			return location, i - 1
		}
		location = next
	}
	return nil, 0
}

func (t *TreeRoot[T]) GetFromIPStr(ipStr string) (UserSuppliedType[T], *net.IPNet, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	var (
		ipaddr net.IP
		err    error
	)
	ipaddr = net.ParseIP(ipStr)
	if ipaddr == nil {
		log.Printf("could not parse IP address=[%s], attempting to parse as CIDR\n", ipStr)
		ipaddr, _, err = net.ParseCIDR(ipStr)
		if err != nil {
			return nil, nil, fmt.Errorf("could not parse IP address=[%s] as IP or CIDR, err=[%s]", ipStr, err)
		}
	}
	return t.GetFromIP(ipaddr)
}

func (t *TreeRoot[T]) GetFromIP(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	v4addr := ipaddr.To4()
	if v4addr != nil {
		return t.getFromIPv4(v4addr)
	} else {
		return t.getFromIPv6(ipaddr)
	}
}

func (t *TreeRoot[T]) getFromIPv4(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	node, i := t.getNodeFromIPv4(ipaddr)
	if node == nil || node.valuePtr == nil {
		return nil, nil, fmt.Errorf("not found")
	}

	network := new(net.IPNet)
	network.IP = ipaddr
	network.Mask = net.CIDRMask(int(i), 32)

	return node.value(), network, nil
}

func (t *TreeRoot[T]) getFromIPv6(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	node, i := t.getNodeFromIPv6(ipaddr)
	if node == nil || node.valuePtr == nil {
		return nil, nil, fmt.Errorf("not found")
	}

	network := new(net.IPNet)
	network.IP = ipaddr
	network.Mask = net.CIDRMask(int(i), 128)

	return node.value(), network, nil
}

// v4bit is a simple function to return the n'th bit of the v4 uint32
func v4bit(v4 uint32, n uint32) uint {
	return uint((v4 >> (32 - n)) & 0x01)
}

func (t *TreeNode[T]) value() UserSuppliedType[T] {
	if t == nil {
		return nil
	}

	if t.valuePtr == nil {
		return nil
	}
	return *t.valuePtr
}

func ipVersion(network *net.IPNet) int {
	_, bitsize := network.Mask.Size()
	switch bitsize {
	case 32:
		return 4
	case 128:
		return 6
	default:
		return 0
	}
}
