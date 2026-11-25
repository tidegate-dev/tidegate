package proxy

import (
	"io"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

const (
	maxVariableNameBytes  = 64
	maxVariableValueBytes = 1024
)

// relay joins the client's session channel to the host's until the host's
// side ends or the client closes its channel. It passes data both ways and
// forwards the requests that allowedFromClient and allowedFromHost name. It
// answers every other request with a failure.
func relay(client ssh.Channel, clientRequests <-chan *ssh.Request, host ssh.Channel, hostRequests <-chan *ssh.Request) {
	var (
		clientDone sync.WaitGroup
		hostDone   sync.WaitGroup
		hostOutput sync.WaitGroup
		answering  sync.Mutex
	)

	clientDone.Go(func() {
		for request := range clientRequests {
			answering.Lock()
			forward(request, host, allowedFromClient(request))
			answering.Unlock()
		}

		_ = host.Close()
	})

	clientDone.Go(func() {
		_, _ = io.Copy(host, client)
		_ = host.CloseWrite()
	})

	hostDone.Go(func() {
		for request := range hostRequests {
			forward(request, client, allowedFromHost(request))
		}
	})

	hostOutput.Go(func() {
		_, _ = io.Copy(client, host)
	})

	hostOutput.Go(func() {
		_, _ = io.Copy(client.Stderr(), host.Stderr())
	})

	// An SSH channel has one end of output for both streams.
	hostDone.Go(func() {
		hostOutput.Wait()

		_ = client.CloseWrite()
	})

	hostDone.Wait()

	// The ssh package doesn't order a close after an answer that is still on its way: https://github.com/golang/go/issues/29733
	answering.Lock()
	_ = client.Close()
	answering.Unlock()

	_ = host.Close()

	clientDone.Wait()
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

// allowedVariable reports if the payload of an env request sets LANG or a
// variable like LC_ALL, with a value of at most maxVariableValueBytes.
func allowedVariable(payload []byte) bool {
	var variable struct{ Name, Value string }

	err := ssh.Unmarshal(payload, &variable)
	if err != nil || len(variable.Name) > maxVariableNameBytes || len(variable.Value) > maxVariableValueBytes {
		return false
	}

	category, isLocale := strings.CutPrefix(variable.Name, "LC_")

	return variable.Name == "LANG" || isLocale && isLocaleCategory(category)
}

func isLocaleCategory(text string) bool {
	if text == "" {
		return false
	}

	for _, character := range []byte(text) {
		isLetter := character >= 'A' && character <= 'Z'
		isDigit := character >= '0' && character <= '9'

		if !isLetter && !isDigit && character != '_' {
			return false
		}
	}

	return true
}
