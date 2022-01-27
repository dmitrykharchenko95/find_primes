// https://docs.google.com/forms/d/e/1FAIpQLSdUMoPoOxIRnVZzlE_Z8tuK9edDXh94bau1-C-C-VFX_oufAQ/viewform - анкета
// https://go.dev/play/p/dRHqCilzyPv - ex1
// https://go.dev/play/p/K9SJf1VrYpT - ex2
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultFile    = "results.txt"
	defaultTimeout = "10"
)

type (
	rangeFlag []string
	limit     struct {
		min int
		max int
	}
)

func (r *rangeFlag) String() string {
	s := strings.Builder{}
	for _, v := range *r {
		s.WriteString(v)
	}
	return s.String()
}

func (r *rangeFlag) Set(value string) error {
	*r = append(*r, strings.TrimSpace(value))
	return nil
}

var (
	FilePath        string
	Timeout         string
	Range           rangeFlag
	ErrWrongTimeout = errors.New("timeout should be int")
	ErrEmptyRange   = errors.New("range shouldn't be empty")
	ErrWrongRange   = errors.New("range should be in the format 'X:Y', where X and Y are int and X<Y")
)

func init() {
	flag.StringVar(&FilePath, "file", defaultFile, "path to the file with found prime numbers")
	flag.StringVar(&Timeout, "timeout", defaultTimeout, "value in seconds, according to the version of the program "+
		"that should finish its execution")
	flag.Var(&Range, "range", "the range of numbers within which the program must find prime numbers")
}

func main() {
	flag.Parse()

	timeout, err := time.ParseDuration(Timeout + "s")
	if err != nil {
		fmt.Println(ErrWrongTimeout)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	wg := sync.WaitGroup{}

	limits, err := parseLimits(Range)
	if err != nil {
		fmt.Println(err)
		return
	}

	wg.Add(len(limits))

	for _, l := range limits {
		go writeToFile(&wg, findPrimes(ctx, l.min, l.max))
	}

	wg.Wait()
}

func parseLimits(args rangeFlag) ([]limit, error) {
	if len(args) == 0 {
		return nil, ErrEmptyRange
	}

	var limits []limit
	var err error

	for _, a := range args {
		var l limit

		val := strings.Split(a, ":")

		if len(val) != 2 {
			return nil, ErrWrongRange
		}

		l.min, err = strconv.Atoi(val[0])
		if err != nil {
			return nil, err
		}

		l.max, err = strconv.Atoi(val[1])
		if err != nil {
			return nil, err
		}

		if l.min >= l.max {
			return nil, ErrWrongRange
		}

		limits = append(limits, l)
	}
	return limits, nil
}

func findPrimes(ctx context.Context, a, b int) <-chan string {
	res := strings.Builder{}
	ch := make(chan string)

	go func() {
		defer close(ch)
		for i := a; i <= b; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				if isPrime(ctx, i) {
					res.WriteString(strconv.Itoa(i) + " ")
				}
			}
		}
		ch <- res.String()
	}()

	return ch
}

func isPrime(ctx context.Context, x int) bool {
	if x == 0 || x == 1 {
		return false
	}
	if x == 2 || x == 3 {
		return true
	}
	for i := 2; i <= x/2; i++ {
		select {
		case <-ctx.Done():
			return false
		default:
			if x%i == 0 {
				return false
			}
		}
	}
	return true
}

func writeToFile(wg *sync.WaitGroup, ch <-chan string) {
	defer wg.Done()

	file, err := os.OpenFile(FilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		fmt.Printf("can not open file: %v", err)
		return
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("can not close file: %v", err)
			return
		}
	}(file)

	_, err = file.WriteString(<-ch)
	if err != nil {
		fmt.Printf("can not write result in file: %v", err)
		return
	}
}
