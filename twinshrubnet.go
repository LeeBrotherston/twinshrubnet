package twinshrubnet

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"net"
	"sync"
)

// Common errors returned by the package
var (
	ErrInvalidIP   = fmt.Errorf("invalid IP address")
	ErrNoMatch     = fmt.Errorf("no matching network found")
	ErrInvalidCIDR = fmt.Errorf("invalid CIDR notation")
)

// UserSuppliedType represents any type that can be stored in the tree
type UserSuppliedType[T any] any

// TreeNode is a node in the binary search tree
type TreeNode[T any] struct {
	binZero *TreeNode[T]
	binOne  *TreeNode[T]
	Value   UserSuppliedType[T]
}

// treeSection represents a versioned tree (IPv4 or IPv6)
type treeSection[T any] struct {
	root *TreeNode[T]
	lock *sync.RWMutex
}

// TreeRoot represents the root of the binary search tree containing both IPv4 and IPv6 trees
type TreeRoot[T any] struct {
	ipv4 *treeSection[T]
	ipv6 *treeSection[T]
}

// NewTree returns the root of a new twinshrubnet tree
func NewTree[T any]() *TreeRoot[T] {
	return &TreeRoot[T]{
		ipv4: &treeSection[T]{
			root: &TreeNode[T]{},
			lock: new(sync.RWMutex),
		},
		ipv6: &treeSection[T]{
			root: &TreeNode[T]{},
			lock: new(sync.RWMutex),
		},
	}
}

// bitGetter interface abstracts IPv4/IPv6 bit operations
type bitGetter interface {
	getBit(position int) uint
	getBitSize() int
}

// ipv4Bits wraps uint32 for bit operations
type ipv4Bits struct {
	addr uint32
}

func (v4 ipv4Bits) getBit(position int) uint {
	// Fix bit position calculation to match test expectations
	// For 192.168.1.1, the first bit should be 1 (192 starts with 11000000)
	return uint((v4.addr >> (32 - uint(position))) & 0x01)
}

func (v4 ipv4Bits) getBitSize() int {
	return 32
}

// ipv6Bits wraps big.Int for bit operations
type ipv6Bits struct {
	addr *big.Int
}

func (v6 ipv6Bits) getBit(position int) uint {
	return uint(v6.addr.Bit(128 - position))
}

func (v6 ipv6Bits) getBitSize() int {
	return 128
}

// createNode creates a new tree node in the specified direction
func createNode[T any](parent *TreeNode[T], isOne bool) *TreeNode[T] {
	node := &TreeNode[T]{}
	if isOne {
		parent.binOne = node
	} else {
		parent.binZero = node
	}
	return node
}

// traverseTree handles the common tree traversal logic
func traverseTree[T any](root *TreeNode[T], bits bitGetter, maskOnes int) (*TreeNode[T], error) {
	location := root
	for i := 1; i <= maskOnes; i++ {
		bit := bits.getBit(i)
		var next *TreeNode[T]
		if bit == 0 {
			if location.binZero == nil {
				next = createNode(location, false)
			} else {
				next = location.binZero
			}
		} else {
			if location.binOne == nil {
				next = createNode(location, true)
			} else {
				next = location.binOne
			}
		}
		location = next
	}
	return location, nil
}

// AddNet adds a network to the tree and returns a pointer to the node representing that network
func (t *TreeRoot[T]) AddNet(cidr string, userdata T) (*TreeNode[T], error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, ErrInvalidCIDR
	}

	maskOnes, bitsize := ipnet.Mask.Size()
	var section *treeSection[T]
	var bits bitGetter

	switch bitsize {
	case 32:
		section = t.ipv4
		bits = ipv4Bits{addr: binary.BigEndian.Uint32(ipnet.IP)}
	case 128:
		section = t.ipv6
		v6 := big.NewInt(0)
		v6.SetBytes(ipnet.IP)
		bits = ipv6Bits{addr: v6}
	default:
		return nil, ErrInvalidIP
	}

	section.lock.Lock()
	defer section.lock.Unlock()

	location, err := traverseTree(section.root, bits, maskOnes)
	if err != nil {
		return nil, err
	}

	location.Value = userdata
	return location, nil
}

// GetFromIP looks up an IP address from its net.IP representation
func (t *TreeRoot[T]) GetFromIP(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	if ipaddr == nil {
		return nil, nil, ErrInvalidIP
	}

	v4addr := ipaddr.To4()
	if v4addr != nil {
		t.ipv4.lock.RLock()
		defer t.ipv4.lock.RUnlock()
		return t.getFromIPv4(v4addr)
	} else {
		t.ipv6.lock.RLock()
		defer t.ipv6.lock.RUnlock()
		return t.getFromIPv6(ipaddr)
	}
}

// GetFromIPStr looks up an IP address from its string representation.
// This method is safe for concurrent use with other methods.
func (t *TreeRoot[T]) GetFromIPStr(ipStr string) (UserSuppliedType[T], *net.IPNet, error) {
	if ipStr == "" {
		return nil, nil, ErrInvalidIP
	}

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

func (t *TreeRoot[T]) getFromIPv4(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	node, net, err := t.getFromIPv4Raw(ipaddr)
	if node == nil {
		return nil, net, err
	}
	return node.Value, net, err

}

func (t *TreeRoot[T]) getFromIPv6(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	node, net, err := t.getFromIPv6Raw(ipaddr)
	if node == nil {
		return nil, net, err
	}
	return node.Value, net, err
}

func (t *TreeRoot[T]) getFromIPv4Raw(ipaddr net.IP) (*TreeNode[T], *net.IPNet, error) {
	if len(ipaddr) != net.IPv4len {
		return nil, nil, ErrInvalidIP
	}

	var network net.IPNet
	location := t.ipv4.root

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
				return location, &network, nil
			}
		}
		location = next
	}
	return nil, nil, fmt.Errorf("no results for search")
}

func (t *TreeRoot[T]) getFromIPv6Raw(ipaddr net.IP) (*TreeNode[T], *net.IPNet, error) {
	if len(ipaddr) != net.IPv6len {
		return nil, nil, ErrInvalidIP
	}

	var network net.IPNet
	location := t.ipv6.root

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
				return location, &network, nil
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
