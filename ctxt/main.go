package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	timeout := time.Second * 1
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		doSomething(ctx)
	}()

	select {
	case <-ctx.Done():
		fmt.Println("context deadline exceeded!")
	}
}

func doSomething(ctx context.Context) {
	fmt.Println("Doing something...")
	time.Sleep(time.Microsecond * 300)
	fmt.Println("Did something...")
}
