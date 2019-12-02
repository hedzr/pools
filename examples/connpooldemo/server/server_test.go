// Copyright © 2019 Hedzr Yeh.

package server

import "testing"

const addr = ":1911"

func TestNewTcpServer(t *testing.T) {
	appExit := make(chan struct{})
	s := New(appExit, addr)
	defer s.Close()
}
