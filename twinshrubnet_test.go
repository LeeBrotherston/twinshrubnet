package twinshrubnet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddAndRetrieveV4(t *testing.T) {
	// Using the type of string to set hello
	myTree := NewTree[string]()

	moo, err := myTree.AddNet("15.197.148.33/18", "Hello")
	require.NoError(t, err)
	require.Equal(t, moo.Value, "Hello")

	resultOne, err := myTree.FindNetFromIP("15.197.148.34")
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
