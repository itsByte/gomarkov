package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/itsByte/gomarkov"
	"github.com/montanaflynn/stats"
)

const minimumProbability = 0.05

type model struct {
	Mean   float64         `json:"mean"`
	StdDev float64         `json:"std_dev"`
	Chain  *gomarkov.Chain `json:"chain"`
}

func main() {
	train := flag.Bool("train", false, "Train the markov chain")
	username := flag.String("u", "", "Username to classify")
	flag.Parse()
	if *train {
		chain, err := buildChain()
		if err != nil {
			fmt.Println(err)
			return
		}
		defer chain.Close()
	} else {
		if len(*username) == 0 {
			flag.Usage()
			return
		}
		chain, err := loadChain()
		if err != nil {
			fmt.Println(err)
			return
		}
		defer chain.Close()
		model, err := buildModel(chain)
		if err != nil {
			fmt.Println(err)
			return
		}
		score := sequenceProbablity(model.Chain, *username)
		normalizedScore := (score - model.Mean) / model.StdDev
		isGibberish := normalizedScore < 0
		fmt.Printf("Score: %f | Gibberish: %t\n", normalizedScore, isGibberish)
	}
}

func buildModel(chain *gomarkov.Chain) (model, error) {
	var model model
	scores := getScores(chain)
	model.StdDev, _ = stats.StandardDeviation(scores)
	model.Mean, _ = stats.Mean(scores)
	model.Chain = chain
	return model, nil
}

func loadChain() (*gomarkov.Chain, error) {
	storage, err := gomarkov.NewPebbleStorage("db")
	if err != nil {
		return nil, err
	}
	return gomarkov.NewChain(2, storage), nil
}

func buildChain() (*gomarkov.Chain, error) {
	chain, err := loadChain()
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	for i, data := range getDataset("usernames.txt") {
		wg.Add(1)
		go func(i int, data string) {
			defer wg.Done()
			chain.Add(1, split(data))
		}(i, data)
	}
	wg.Wait()
	return chain, nil
}

func getScores(chain *gomarkov.Chain) []float64 {
	scores := make([]float64, 0)
	for _, data := range getDataset("train.txt") {
		score := sequenceProbablity(chain, data)
		scores = append(scores, score)
	}
	return scores
}

func getDataset(fileName string) []string {
	file, _ := os.Open(fileName)
	scanner := bufio.NewScanner(file)
	var list []string
	for scanner.Scan() {
		list = append(list, scanner.Text())
	}
	return list
}

func split(str string) []string {
	return strings.Split(str, "")
}

func sequenceProbablity(chain *gomarkov.Chain, input string) float64 {
	tokens := split(input)
	logProb := float64(0)
	pairs := gomarkov.MakePairs(tokens, chain.Order)
	for _, pair := range pairs {
		prob, _ := chain.TransitionProbability(1, pair.NextState, pair.CurrentState)
		if prob > 0 {
			logProb += math.Log10(prob)
		} else {
			logProb += math.Log10(minimumProbability)
		}
	}
	return math.Pow(10, logProb/float64(len(pairs)))
}
