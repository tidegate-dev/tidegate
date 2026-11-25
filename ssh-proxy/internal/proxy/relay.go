package proxy

import (
	"io"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

const maxVariableValueBytes = 1024

// relay joins the client's session channel to the host's until the host's
// side ends or the client closes its channel. It passes data both ways and
// forwards the requests that allowedFromClient and allowedFromHost name. It
// answers every other request with a failure.
func relay(client ssh.Channel, clientRequests <-chan *ssh.Request, host ssh.Channel, hostRequests <-chan *ssh.Request) {
	var (
		hostDone  sync.WaitGroup
		answering sync.Mutex
	)

	go func() {
		for request := range clientRequests {
			answering.Lock()
			forward(request, host, allowedFromClient(request))
			answering.Unlock()
		}

		_ = host.Close()
	}()

	go func() {
		_, _ = io.Copy(host, client)
		_ = host.CloseWrite()
	}()

	hostDone.Go(func() {
		for request := range hostRequests {
			forward(request, client, allowedFromHost(request))
		}
	})

	hostDone.Go(func() {
		_, _ = io.Copy(client, host)
	})

	hostDone.Go(func() {
		_, _ = io.Copy(client.Stderr(), host.Stderr())
	})

	hostDone.Wait()

	// The ssh package doesn't order a close after an answer that is still on its way: https://github.com/golang/go/issues/29733
	answering.Lock()
	_ = client.Close()
	answering.Unlock()

	_ = host.Close()
}

// forward sends an allowed request to the other side and answers it with the
// other side's answer.
func forward(request *ssh.Request, to ssh.Channel, allowed bool) {
	ok := false

	if allowed {
		answer, err := to.SendRequest(request.Type, request.WantReply, request.Payload)
		ok = err == nil && answer
	}

	if request.WantReply {
		_ = request.Reply(ok, nil)
	}
}

func allowedFromClient(request *ssh.Request) bool {
	switch request.Type {
	case "pty-req", "shell", "exec", "window-change":
		return true
	case "env":
		return allowedVariable(request.Payload)
	default:
		return false
	}
}

func allowedFromHost(request *ssh.Request) bool {
	return request.Type == "exit-status" || request.Type == "exit-signal"
}

func allowedVariable(payload []byte) bool {
	var variable struct{ Name, Value string }

	err := ssh.Unmarshal(payload, &variable)
	if err != nil || len(variable.Value) > maxVariableValueBytes {
		return false
	}

	return variable.Name == "LANG" || strings.HasPrefix(variable.Name, "LC_")
}
