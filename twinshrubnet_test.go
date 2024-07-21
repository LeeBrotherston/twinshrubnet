package twinshrubnet

import (
	"encoding/binary"
	"net"
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
	t.Logf("something: %+v %+v\n", &moo, moo)
	require.NoError(t, err)
	require.Equal(t, moo.Value, "Hello")

	resultOne, _ := myTree.FindNetFromIP("10.10.10.3")
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

	resultTwo, err := myTree.FindNetFromIP("bd5f:285d:2687:ec0c:0000:0000:0000:0001")
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

	result, err := myTree.FindNetFromIP("192.168.5.34")
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

	result, err := myTree.FindNetFromIP("10.10.10.10")
	require.NoError(t, err)
	require.Nil(t, result)
}
