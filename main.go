package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	circuit "github.com/cep21/circuit/v3"
	cepHystrix "github.com/cep21/circuit/v3/closers/hystrix"
	"github.com/zenthangplus/goccm"
)

func main() {
	useCep := false

	if useCep {
		configuration := cepHystrix.Factory{
			// Hystrix open logic is to open the circuit after an % of errors
			ConfigureOpener: cepHystrix.ConfigureOpener{
				// We change the default to wait for 20 requests, before checking to close (default)
				RequestVolumeThreshold:   20,
				ErrorThresholdPercentage: 50,
			},
			// Hystrix close logic is to sleep then check
			ConfigureCloser: cepHystrix.ConfigureCloser{
				SleepWindow: time.Millisecond * 3,
			},
		}

		h := circuit.Manager{
			DefaultCircuitProperties: []circuit.CommandPropertiesConstructor{configuration.Configure},
		}
		c := h.MustCreateCircuit("test")

		counter := 0
		start := time.Now()

		for i := 0; i < 300; i++ {
			doCep21Circuit(i, &counter, c)
		}

		fmt.Println(time.Since(start).Milliseconds())
		fmt.Println("total", counter)
	} else {
		hystrix.ConfigureCommand("test", hystrix.CommandConfig{
			SleepWindow:            100,
			Timeout:                1000,
			MaxConcurrentRequests:  1,
			ErrorPercentThreshold:  100,
			RequestVolumeThreshold: 50,
		})

		main := make(chan bool, 10000)
		backup := make(chan bool, 10000)
		openCircuit := make(chan bool, 10000)

		start := time.Now()
		wg := sync.WaitGroup{}
		wg.Add(300)

		c := goccm.New(20)

		for i := 0; i < 300; i++ {
			c.Wait()

			go func() {
				doHystrix(i, main, backup, openCircuit)
				wg.Done()

				c.Done()
			}()
		}

		c.WaitAllDone()
		wg.Wait()

		close(main)
		close(backup)
		close(openCircuit)

		totalMain := 0
		for range main {
			totalMain++
		}

		totalBackup := 0
		for range backup {
			totalBackup++
		}

		totalOpenCircuit := 0
		for range openCircuit {
			totalOpenCircuit++
		}

		fmt.Println(time.Since(start).Milliseconds())
		fmt.Println("totalMain", totalMain)
		fmt.Println("totalBackup", totalBackup)
		fmt.Println("totalOpenCircuit", totalOpenCircuit)
	}

}

func doHystrix(i int, main chan bool, backup chan bool, openCircuit chan bool) {
	var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	time.Sleep(time.Nanosecond * 2)

	err := hystrix.DoC(context.Background(), "test", func(ctx context.Context) error {
		errorPercentage := 25

		// if i > 50 && i <= 150 {
		// 	errorPercentage = 75
		// }

		fmt.Println("errorPercentage", errorPercentage, "i", i)

		x := rng.Intn(100)
		if x <= errorPercentage {
			fmt.Printf("erroring out in initial call\n")
			return errors.New("test")
		}

		fmt.Println("success in initial call")
		main <- true
		return nil
	}, func(ctx context.Context, err error) error {
		fmt.Printf("In backup: %s\n", err.Error())

		if err.Error() == "hystrix: circuit open" {
			openCircuit <- true
		}

		backup <- true
		return nil
	})

	if err != nil {
		fmt.Printf("error from circuit breaker: %v\n", err.Error())
	}
}

func doCep21Circuit(i int, counter *int, c *circuit.Circuit) {
	var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	errResult := c.Execute(context.Background(), func(ctx context.Context) error {
		errorPercentage := 25

		if i > 50 && i <= 150 {
			errorPercentage = 75
		}

		fmt.Println("errorPercentage", errorPercentage, "i", i)

		x := rng.Intn(100)
		if x <= errorPercentage {
			fmt.Printf("erroring out in initial call\n")
			return errors.New("test")
		}

		fmt.Println("success in initial call")
		// *counter++
		return nil
	}, func(ctx context.Context, err error) error {
		fmt.Printf("In backup: %s\n", err.Error())
		// *counter++
		return nil
	})
	fmt.Println("Execution result:", errResult)
}
