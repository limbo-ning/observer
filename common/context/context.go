package context

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var ctx context.Context
var lock sync.RWMutex
var cancels = make(map[context.Context]context.CancelFunc)

var wg sync.WaitGroup

func init() {
	ctx = context.Background()

	if ctx == nil {
		panic("background ctx nil")
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSEGV)

	go func() {
		log.Println("background context listening signal")
		s := <-sig
		ctx = nil
		log.Println("receive signal. exiting: ", s.String())
		log.Println("child contexts to cancel: ", len(cancels))

		lock.Lock()
		defer lock.Unlock()

		for child, cancel := range cancels {
			log.Println("cancel child ctx: ", child)
			cancel()
		}

		log.Println("wait group waiting")
		wg.Wait()
		log.Println("gracefully exit")
		os.Exit(0)
	}()
}

func addContext(child context.Context, cancel func()) (context.Context, func()) {
	lock.Lock()
	cancels[child] = cancel
	wg.Add(1)
	lock.Unlock()

	return child, func() {
		lock.Lock()
		if _, exists := cancels[child]; exists {
			delete(cancels, child)
			cancel()
			wg.Done()
		} else {
			log.Println("child ctx not exists")
		}
		lock.Unlock()
	}
}

func GetContext() (context.Context, func()) {

	if ctx == nil {
		log.Println("error get context: ctx nil. should be exiting.")
		select {}
	}

	return addContext(context.WithCancel(ctx))
}

func GetContextWithDeadline(timeout time.Duration) (context.Context, func()) {
	if ctx == nil {
		log.Println("error get context: ctx nil should be exiting.")
		select {}
	}

	return addContext(context.WithTimeout(ctx, timeout))
}
