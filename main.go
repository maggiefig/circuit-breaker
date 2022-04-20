package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	circuit "github.com/cep21/circuit/v3"
	cepHystrix "github.com/cep21/circuit/v3/closers/hystrix"
)

func main() {
	useCep := true

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
			SleepWindow:           3,
			Timeout:               1000,
			MaxConcurrentRequests: 100,
			ErrorPercentThreshold: 50,
		})

		counter := 0
		start := time.Now()

		for i := 0; i < 300; i++ {
			doHystrix(i, &counter)
		}

		fmt.Println(time.Since(start).Milliseconds())
		fmt.Println("total", counter)
	}

}

func doHystrix(i int, counter *int) {
	var rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	err := hystrix.DoC(context.Background(), "test", func(ctx context.Context) error {
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
		*counter++
		return nil
	}, func(ctx context.Context, err error) error {
		fmt.Printf("In backup: %s\n", err.Error())
		*counter++
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
		*counter++
		return nil
	}, func(ctx context.Context, err error) error {
		fmt.Printf("In backup: %s\n", err.Error())
		*counter++
		return nil
	})
	fmt.Println("Execution result:", errResult)
}
