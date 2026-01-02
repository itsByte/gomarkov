package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/itsByte/gomarkov"
)

func main() {
	train := flag.Bool("train", false, "Train the markov chain")
	order := flag.Int("order", 3, "Chain order to use")
	flag.Parse()
	if *train {
		chain, err := buildModel(*order)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer chain.Close()
	} else {
		chain, err := loadChain(*order)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer chain.Close()
		generatePokemon(chain)
	}
}

func buildModel(order int) (*gomarkov.Chain, error) {
	chain, err := loadChain(order)
	if err != nil {
		return nil, err
	}
	for _, data := range getDataset("names.txt") {
		chain.Add(1, split(data))
	}
	return chain, nil
}

func split(str string) []string {
	return strings.Split(str, "")
}

func getDataset(fileName string) []string {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var list []string
	for scanner.Scan() {
		list = append(list, scanner.Text())
	}
	return list
}

func loadChain(order int) (*gomarkov.Chain, error) {
	storage, err := gomarkov.NewPebbleStorage("db")
	if err != nil {
		return nil, err
	}
	return gomarkov.NewChain(order, storage), nil
}

func generatePokemon(chain *gomarkov.Chain) {
	order := chain.Order
	tokens := make([]string, 0)
	for range order {
		tokens = append(tokens, gomarkov.StartToken)
	}
	for tokens[len(tokens)-1] != gomarkov.EndToken {
		next, _ := chain.Generate(1, tokens[(len(tokens)-order):])
		tokens = append(tokens, next)
	}
	fmt.Println(strings.Join(tokens[order:len(tokens)-1], ""))
}
