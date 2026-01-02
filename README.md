# gomarkov
[![GoDoc](https://godoc.org/github.com/itsbyte/gomarkov?status.svg)](https://godoc.org/github.com/mb-14/gomarkov)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsbyte/gomarkov)](https://goreportcard.com/report/github.com/mb-14/gomarkov)

Go implementation of markov chains for textual data.

You can find out more about markov chains [here](http://setosa.io/ev/markov-chains/) and [here](https://towardsdatascience.com/introduction-to-markov-chains-50da3645a50d)

## Usage
```go
package main

import (
	"fmt"
	"strings"

	"github.com/itsByte/gomarkov"
)

func main() {
	//Initialize Pebble
	storage, err := gomarkov.NewPebbleStorage("db")
	if err != nil {
		fmt.Println(err)
	}

	//Create a chain of order 2
	chain := gomarkov.NewChain(2, storage)

	//Feed in training data, specifying the context id you want to use
	chain.Add(1, strings.Split("I want a cheese burger", " "))
	chain.Add(1, strings.Split("I want a chilled sprite", " "))
	chain.Add(1, strings.Split("I want to go to the movies", " "))

	//Get transition probability of a sequence using the context id we specified earlier
	prob, _ := chain.TransitionProbability(1, "a", []string{"I", "want"})
	fmt.Println(prob)
	//Output: 0.6666666666666666

	//You can even generate new text based on an initial seed
	chain.Add(2, strings.Split("Mother should I build the wall?", " "))
	chain.Add(2, strings.Split("Mother should I run for President?", " "))
	chain.Add(2, strings.Split("Mother should I trust the government?", " "))
	next, _ := chain.Generate(2, []string{"should", "I"})
	fmt.Println(next)
}

```
## Examples

- [Gibberish username detector](/examples/gibberish)

- [Fake Hackernews post generator](/examples/fakernews)

- [Pokemon name generator](/examples/pokenamer)
