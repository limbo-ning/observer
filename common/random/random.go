package random

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

var intOutput = make(chan int)
var intInput = make(chan int)

func init() {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	go func() {
		for {
			func() {
				if err := recover(); err != nil {
					log.Println("error random: ", err)
					random = rand.New(rand.NewSource(time.Now().UnixNano()))
				}
				intOutput <- random.Intn(<-intInput)
			}()
		}
	}()
}

func GenerateNonce(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		intInput <- 52
		r := <-intOutput
		if r < 26 {
			result = fmt.Sprintf("%s%c", result, 'a'+r)
		} else {
			result = fmt.Sprintf("%s%c", result, 'A'+r-26)
		}
	}

	return result
}

func GetRandomNumber(n int) int {
	if n == 0 {
		return 0
	}
	intInput <- n
	return <-intOutput
}

func GenerateRandomNumber(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		intInput <- 10
		r := <-intOutput
		result += fmt.Sprintf("%d", r)
	}

	return result
}
