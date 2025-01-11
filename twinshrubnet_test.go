package twinshrubnet

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type ipbinmap struct {
	ipStr  string
	binary []uint
}

func TestV4Bit(t *testing.T) {
	tests := []ipbinmap{
		{
			ipStr:  "8.8.8.8",
			binary: []uint{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		}, {
			ipStr:  "0.0.0.1",
			binary: []uint{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		}}

	for _, test := range tests {
		v4Uint32 := binary.BigEndian.Uint32(net.ParseIP(test.ipStr).To4())
		t.Logf("ipstr: %s\nv4Uint32-bin: %b\nbin: %+v", test.ipStr, v4Uint32, test.binary)
		for i := uint32(1); i <= 32; i++ {
			retval := v4bit(v4Uint32, i)
			require.Equal(t, test.binary[i-1], retval)
		}

	}
}

func TestAddV4(t *testing.T) {
	// Using the type of string to set hello
	myTree := NewTree[string]()

	moo, err := myTree.AddNet("10.10.10.1/18", "Hello")
	require.NoError(t, err)
	require.Equal(t, moo.Value, "Hello")
}

func TestAddV6(t *testing.T) {
	// Using the type of string to set hello
	myTree := NewTree[string]()

	something, err := myTree.AddNet("bd5f:285d:2687:ec0c:0a3b:9f7a:cb63:560b/64", "yo yo yo")
	require.NoError(t, err)
	require.Equal(t, something.Value, "yo yo yo")
}

func TestAddAndRetrieveV4(t *testing.T) {
	// Using the type of string to set hello
	myTree := NewTree[string]()

	moo, err := myTree.AddNet("10.10.10.1/18", "Hello")
	require.NoError(t, err)
	require.Equal(t, moo.Value, "Hello")

	resultOne, network, _ := myTree.GetFromIPStr("10.10.10.3")
	netsize, _ := network.Mask.Size()
	require.Equal(t, 18, netsize)
	require.NoError(t, err)
	require.NotNil(t, resultOne)
	require.Equal(t, "Hello", resultOne)
}

func TestAddAndRetrieveV6(t *testing.T) {
	// Using the type of string to set hello
	myTree := NewTree[string]()

	something, err := myTree.AddNet("bd5f:285d:2687:ec0c:0a3b:9f7a:cb63:560b/64", "yo yo yo")
	require.NoError(t, err)
	require.Equal(t, something.Value, "yo yo yo")

	resultTwo, network, err := myTree.GetFromIPStr("bd5f:285d:2687:ec0c:0000:0000:0000:0001")
	netsize, _ := network.Mask.Size()
	require.Equal(t, 64, netsize)
	require.NoError(t, err)
	require.NotNil(t, resultTwo)
	require.Equal(t, "yo yo yo", resultTwo)
}

func TestOverlapV4(t *testing.T) {
	// Using the type of string to set hello
	myTree := NewTree[string]()

	mooOne, err := myTree.AddNet("192.168.1.0/16", "Larger")
	require.NoError(t, err)
	require.Equal(t, mooOne.Value, "Larger")

	mooTwo, err := myTree.AddNet("192.168.5.0/24", "Smaller")
	require.NoError(t, err)
	require.Equal(t, mooTwo.Value, "Smaller")

	result, network, err := myTree.GetFromIPStr("192.168.5.34")
	netsize, _ := network.Mask.Size()
	require.Equal(t, 24, netsize)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "Smaller", result)
}

func TestNotFoundV4(t *testing.T) {
	// Using the type of string to set hello
	myTree := NewTree[string]()

	mooOne, err := myTree.AddNet("192.168.1.0/16", "Larger")
	require.NoError(t, err)
	require.Equal(t, mooOne.Value, "Larger")

	mooTwo, err := myTree.AddNet("192.168.5.0/24", "Smaller")
	require.NoError(t, err)
	require.Equal(t, mooTwo.Value, "Smaller")
	require.Equal(t, mooOne.Value, "Larger")

	result, network, err := myTree.GetFromIPStr("10.10.10.10")
	require.Nil(t, network)
	require.Nil(t, result)
	require.NoError(t, err)
}

func TestSingleNodeNetV4(t *testing.T) {
	// Using the type of string to set hello
	myTree := NewTree[string]()

	mooOne, err := myTree.AddNet("192.168.1.2/32", "My thing")
	require.NoError(t, err)
	require.Equal(t, mooOne.Value, "My thing")

	result, _, err := myTree.GetFromIPStr("192.168.1.2")
	require.NoError(t, err)
	require.Equal(t, result, "My thing")
}

func TestInvalidInputs(t *testing.T) {
	myTree := NewTree[string]()

	// Test invalid CIDR
	_, err := myTree.AddNet("invalid", "test")
	require.Error(t, err)

	// Test empty IP
	result, net, err := myTree.GetFromIPStr("")
	require.Error(t, err)
	require.Nil(t, result)
	require.Nil(t, net)

	// Test nil IP
	result, net, err = myTree.GetFromIP(nil)
	require.Error(t, err)
	require.Nil(t, result)
	require.Nil(t, net)
}

func TestBitGetters(t *testing.T) {
	// Test IPv4 bits (192.168.1.1 = 11000000.10101000.00000001.00000001)
	v4 := ipv4Bits{addr: binary.BigEndian.Uint32(net.ParseIP("192.168.1.1").To4())}
	require.Equal(t, uint(1), v4.getBit(1)) // First bit of 192 (1)
	require.Equal(t, uint(1), v4.getBit(2)) // Second bit of 192 (1)
	require.Equal(t, uint(0), v4.getBit(3)) // Third bit of 192 (0)

	// Test IPv6 bits (2001:db8::1 = 0010 0000 0000 0001:...)
	addr := big.NewInt(0)
	addr.SetBytes(net.ParseIP("2001:db8::1"))
	v6 := ipv6Bits{addr: addr}
	require.Equal(t, uint(0), v6.getBit(1)) // First bit (0)
	require.Equal(t, uint(0), v6.getBit(2)) // Second bit (0)
	require.Equal(t, uint(1), v6.getBit(3)) // Third bit (1)
	require.Equal(t, uint(0), v6.getBit(4)) // Fourth bit (0)
}

func TestConcurrentAccess(t *testing.T) {
	myTree := NewTree[string]()
	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 4) // 2 writers + 2 readers for each IP version

	// Add some initial networks
	_, err := myTree.AddNet("192.168.0.0/16", "IPv4 Network")
	require.NoError(t, err)
	_, err = myTree.AddNet("2001:db8::/32", "IPv6 Network")
	require.NoError(t, err)

	// Concurrent IPv4 operations
	for i := 0; i < goroutines; i++ {
		// Writer goroutine
		go func(num int) {
			defer wg.Done()
			_, err := myTree.AddNet(fmt.Sprintf("192.168.%d.0/24", num%256), "IPv4 Subnet")
			require.NoError(t, err)
		}(i)

		// Reader goroutine
		go func(num int) {
			defer wg.Done()
			result, _, err := myTree.GetFromIPStr(fmt.Sprintf("192.168.%d.1", num%256))
			require.NoError(t, err)
			if result != nil {
				require.Contains(t, []string{"IPv4 Network", "IPv4 Subnet"}, result)
			}
		}(i)
	}

	// Concurrent IPv6 operations
	for i := 0; i < goroutines; i++ {
		// Writer goroutine
		go func(num int) {
			defer wg.Done()
			_, err := myTree.AddNet(fmt.Sprintf("2001:db8:%d::/48", num%65536), "IPv6 Subnet")
			require.NoError(t, err)
		}(i)

		// Reader goroutine
		go func(num int) {
			defer wg.Done()
			result, _, err := myTree.GetFromIPStr(fmt.Sprintf("2001:db8:%d::1", num%65536))
			require.NoError(t, err)
			if result != nil {
				require.Contains(t, []string{"IPv6 Network", "IPv6 Subnet"}, result)
			}
		}(i)
	}

	wg.Wait()
}
