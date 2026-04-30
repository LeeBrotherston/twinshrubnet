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

type TreeWrapper struct {
	Root TreeRoot[any]
}

type LookupTree interface {
	AddNet(cidr string, userdata any) (*TreeNode[any], error)
	GetFromIP(ipaddr net.IP) (UserSuppliedType[any], *net.IPNet, error)
	GetFromIPStr(ipaddr string) (UserSuppliedType[any], *net.IPNet, error)
	RemoveNet(cidr string) error
}

// UserSuppliedType represents any type that can be stored in the tree
type UserSuppliedType[T any] any

// TreeNode is a node in the binary search tree
type TreeNode[T any] struct {
	binZero  *TreeNode[T]
	binOne   *TreeNode[T]
	valuePtr *UserSuppliedType[T]
}

// treePartition represents partition from the root of the tree. In this case
// ipv4 and ipv6 as we wish to keep them separate
type treePartition[T any] struct {
	root *TreeNode[T]
	lock *sync.RWMutex
}

// TreeRoot represents the root of the binary search tree containing each
// partition
type TreeRoot[T any] struct {
	ipv4 *treePartition[T]
	ipv6 *treePartition[T]
}

// NewTree returns the root of a new twinshrubnet tree, populated with an ipv4
// and ipv6 partition
func NewTree[T any]() *TreeRoot[T] {
	return &TreeRoot[T]{
		ipv4: &treePartition[T]{
			root: &TreeNode[T]{},
			lock: new(sync.RWMutex),
		},
		ipv6: &treePartition[T]{
			root: &TreeNode[T]{},
			lock: new(sync.RWMutex),
		},
	}
}

// bitGetter interface abstracts IPv4/IPv6 bit operations to make code easier
// elsewhere
type bitGetter interface {
	getBit(position int) uint
	getBitSize() int
}

// ipv4Bits wraps uint32 for bit operations
type ipv4Bits struct {
	addr uint32
}

// ipv6Bits wraps big.Int for bit operations
type ipv6Bits struct {
	addr *big.Int
}

// getBit get the bit in the supplied position in an ipv4 address
func (v4 ipv4Bits) getBit(position int) uint {
	// Ensure position is within valid range (1-32)
	if position < 1 || position > 32 {
		return 0
	}

	// Use uint32 for all arithmetic to avoid overflow
	pos := uint32(position)
	if pos > 32 {
		return 0
	}

	// Calculate shift using uint32
	shift := uint32(32) - pos

	// Perform bit operation using uint32 and only convert final result
	return uint((v4.addr >> shift) & uint32(1))
}

// getBit get the bit in the supplied position in an ipv6 address
func (v6 ipv6Bits) getBit(position int) uint {
	return uint(v6.addr.Bit(128 - position))
}

func (v4 ipv4Bits) getBitSize() int {
	return 32
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
	var partition *treePartition[T]
	var bits bitGetter

	switch bitsize {
	case 32:
		partition = t.ipv4
		bits = ipv4Bits{addr: binary.BigEndian.Uint32(ipnet.IP)}
	case 128:
		partition = t.ipv6
		v6 := big.NewInt(0)
		v6.SetBytes(ipnet.IP)
		bits = ipv6Bits{addr: v6}
	default:
		return nil, ErrInvalidIP
	}

	partition.lock.Lock()
	defer partition.lock.Unlock()

	location, err := traverseTree(partition.root, bits, maskOnes)
	if err != nil {
		return nil, err
	}

	location.valuePtr = new(UserSuppliedType[T])
	*location.valuePtr = userdata

	return location, nil
}

// GetFromIP looks up an IP address from its net.IP representation. This is safe
// for use in concurrent functions
func (t *TreeRoot[T]) GetFromIP(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	if ipaddr == nil {
		return nil, nil, ErrInvalidIP
	}

	v4addr := ipaddr.To4()
	if v4addr != nil {
		return t.getFromIPv4(v4addr)
	} else {
		return t.getFromIPv6(ipaddr)
	}
}

// GetFromIPStr looks up an IP address from its string representation.  This is
// safe for use in concurrent functions
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

// getFromIPv4 is the ipv4 specific search function
func (t *TreeRoot[T]) getFromIPv4(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	if len(ipaddr) != net.IPv4len {
		return nil, nil, ErrInvalidIP
	}

	// Ensure we have a read lock so that we don't traverse a tree that's being
	// modified
	t.ipv4.lock.RLock()
	defer t.ipv4.lock.RUnlock()

	location := t.ipv4.root
	v4Uint32 := binary.BigEndian.Uint32(ipaddr)
	var i uint32

	for i = 1; i < 34; i++ {
		thing := v4bit(v4Uint32, i)
		var next *TreeNode[T]
		if thing == 0 {
			next = location.binZero
		} else {
			next = location.binOne
		}

		if next == nil {
			if location.valuePtr == nil {
				return nil, nil, ErrNoMatch
			}
			network := &net.IPNet{
				IP:   ipaddr,
				Mask: net.CIDRMask(int(i-1), 32),
			}
			return location.value(), network, nil
		}
		location = next
	}

	if location.valuePtr == nil {
		return nil, nil, ErrNoMatch
	}
	network := &net.IPNet{
		IP:   ipaddr,
		Mask: net.CIDRMask(int(i-1), 32),
	}
	return location.value(), network, nil
}

// getFromIPv6 is the ipv4 specific search function
func (t *TreeRoot[T]) getFromIPv6(ipaddr net.IP) (UserSuppliedType[T], *net.IPNet, error) {
	if len(ipaddr) != net.IPv6len {
		return nil, nil, ErrInvalidIP
	}

	// Ensure we have a read lock so that we don't traverse a tree that's being
	// modified
	t.ipv6.lock.RLock()
	defer t.ipv6.lock.RUnlock()

	location := t.ipv6.root
	v6 := big.NewInt(0)
	v6.SetBytes(ipaddr)
	var i int

	for i = 1; i <= 128; i++ {
		thing := v6.Bit(128 - i)
		var next *TreeNode[T]
		if thing == 0 {
			next = location.binZero
		} else {
			next = location.binOne
		}

		if next == nil {
			if location.valuePtr == nil {
				return nil, nil, ErrNoMatch
			}
			network := &net.IPNet{
				IP:   ipaddr,
				Mask: net.CIDRMask(i-1, 128),
			}
			return location.value(), network, nil
		}
		location = next
	}

	if location.valuePtr == nil {
		return nil, nil, ErrNoMatch
	}
	network := &net.IPNet{
		IP:   ipaddr,
		Mask: net.CIDRMask(i-1, 128),
	}
	return location.value(), network, nil
}

// Removed getFromIPv4Raw and getFromIPv6Raw as they're no longer needed

// RemoveNet removes a network from the tree
func (t *TreeRoot[T]) RemoveNet(cidr string) error {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	var partition *treePartition[T]
	switch ipVersion(ipnet) {
	case 4:
		partition = t.ipv4
	case 6:
		partition = t.ipv6
	default:
		return ErrInvalidIP
	}

	// We don't need to lock partitions, because findNodeFromIPNet already does
	// this

	node, err := t.findNodeFromIPNet(*ipnet)
	if err != nil {
		return err
	}

	partition.lock.Lock()
	node.valuePtr = nil
	partition.lock.Unlock()
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
	location := t.ipv4.root
	v4Uint32 := binary.BigEndian.Uint32(ipaddr)

	t.ipv4.lock.RLock()
	defer t.ipv4.lock.RUnlock()

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
	location := t.ipv6.root
	v6 := big.NewInt(0)
	v6.SetBytes(ipaddr)

	t.ipv6.lock.RLock()
	defer t.ipv6.lock.RUnlock()

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
