# Binary Tree Subnet Search ... a twin shrubnet search, in GO :)
![Build Pass/Fail Badge](https://github.com/LeeBrotherston/twinshrubnet/actions/workflows/go.yml/badge.svg)

Looking up an IP from a list is easy, looking up if an IP is within a subnet is easy. Having a list of subnets and finding which one the IP address belongs to is less easy. This is a simple binary tree search package which makes this pretty easy, as subnet masking lends itself really nicely to binary tree search.

twinshrubnet uses generics in order to allow user supplied types to be used as values in the lookup and so GO version `1.18` or above is required.

Here's an example:

```golang
    // initialize the tree using type string as the value stored for this subnet
    myTree := twinshrubnet.NewTree[string]()

    // insert a new subnet into the lookup tree
    _, err := myTree.AddNet("15.197.148.33/18", "I am a lookup value")
    if err != nil {
        // handle error
    }

    // Lookup a single IP in that subnet
    someResult, network, err := myTree.FindNetFromIP("15.197.148.34")
    if err != nil {
        // handle error
    }

    // Result and network result was found in can be used
    fmt.Printf("The result is: %s\n", someResult)
    fmt.Printf("Found in network: %s\n", network.String())
```

With overlapping subnets, the most specific is returned.  e.g. a `/30` which is within a `/24` both being stored as subnets... a search for an address inside the `/30` would return the `/30` not the `/24` as this is the most specific subnet.
