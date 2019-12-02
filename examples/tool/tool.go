// Copyright © 2019 Hedzr Yeh.

package tool

import (
	"math/rand"
)

// // SignalToSelf trigger the sig signal to the current process
// func SignalToSelf(sig os.Signal) (err error) {
// 	var p *os.Process
// 	if p, err = os.FindProcess(os.Getpid()); err != nil {
// 		log.Fatalf("can't find process with pid=%v: %v", os.Getpid(), err)
// 	}
// 	err = p.Signal(sig)
// 	return
// }
//
// // SignalQuitSignal post a SIGQUIT signal to the current process
// func SignalQuitSignal() {
// 	_ = SignalToSelf(syscall.SIGQUIT)
// }
//
// // SignalTermSignal post a SIGTERM signal to the current process
// func SignalTermSignal() {
// 	_ = SignalToSelf(syscall.SIGTERM)
// }

// RandomStringPure returns a random characters string for you.
func RandomStringPure(length int) (result string) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err == nil {
		result = string(buf)
	}
	return
}
