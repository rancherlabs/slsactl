package spinner

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

var frames = []rune{'|', '/', '-', '\\'}

const (
	sleep       = 100 * time.Millisecond
	paddingSize = 130

	successIcon = "✅"
	successText = "DONE"
	failureIcon = "❌"
	failureText = "FAILED"
)

type Spinner struct {
	message    string
	statusChan chan string
	wg         sync.WaitGroup
	stop       chan struct{}
}

func New(message string) *Spinner {
	return &Spinner{
		message:    message,
		stop:       make(chan struct{}),
		statusChan: make(chan string),
	}
}

func (s *Spinner) Start() {
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		i := 0
		previousLen := 0
		msg := s.message

		for {
			select {
			case <-s.stop:
				return
			case status := <-s.statusChan:
				msg = s.message + ": " + status
			default:
				newLen := fmt.Sprintf("%c %s...", frames[i%len(frames)], msg)

				padding := ""
				if previousLen > len(newLen) {
					padding = strings.Repeat(" ", previousLen-len(newLen))
				}

				fmt.Printf("\r%s%s", newLen, padding)

				i++
				time.Sleep(sleep)
				previousLen = len(newLen)
			}
		}
	}()
}

func (s *Spinner) UpdateStatus(newStatus string) {
	s.statusChan <- newStatus
}

func (s *Spinner) Stop(success bool) {
	close(s.stop)
	s.wg.Wait()

	clearLine := "\r" + strings.Repeat(" ", paddingSize)
	fmt.Print(clearLine)

	if success {
		fmt.Printf("\r%s %s: %s\n", successIcon, s.message, successText)
	} else {
		fmt.Printf("\r%s %s: %s\n", failureIcon, s.message, failureText)
	}
}
