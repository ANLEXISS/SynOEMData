package network

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type recordingHostCommands struct {
	calls  []string
	output []byte
	err    error
	stdin  string
}

func (r *recordingHostCommands) Run(name string, args ...string) error {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	if name == "ip" && len(args) > 2 && args[0] == "link" && args[1] == "show" {
		return errors.New("bridge missing")
	}
	return nil
}

func (r *recordingHostCommands) Output(name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	return r.output, r.err
}

func (r *recordingHostCommands) RunWithInput(name string, input io.Reader, args ...string) error {
	body, err := io.ReadAll(input)
	if err != nil {
		return err
	}
	r.stdin = string(body)
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	return r.err
}

func TestBridgeUsesInjectedHostCommandExecutor(t *testing.T) {
	commands := &recordingHostCommands{}
	if err := ensureBridgeWithExecutor(DefaultConfig().SynoraNet, commands); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"ip link show synorabr0",
		"ip link add name synorabr0 type bridge",
		"ip addr flush dev synorabr0",
		"ip addr add " + DefaultGateway + "/24 dev synorabr0",
		"ip link set synorabr0 up",
	}
	if !reflect.DeepEqual(commands.calls, want) {
		t.Fatalf("host calls=%q, want %q", commands.calls, want)
	}
}

func TestFirewallExecutorReceivesRenderedInput(t *testing.T) {
	commands := &recordingHostCommands{}
	input := "add table inet synora_net\n"
	if err := runFirewallCommandWithExecutor(commands, "nft", []string{"-f", "-"}, input); err != nil {
		t.Fatal(err)
	}
	if commands.stdin != input || !reflect.DeepEqual(commands.calls, []string{"nft -f -"}) {
		t.Fatalf("calls=%q stdin=%q", commands.calls, commands.stdin)
	}
}
