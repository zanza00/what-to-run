package core

import (
	"reflect"
	"testing"
)

func TestTaskCommand(t *testing.T) {
	task := Task{Args: []string{"make", "build", "GOFLAGS=-v"}, Dir: "ops"}

	cmd := task.Command()
	if got, want := cmd.Args, []string{"make", "build", "GOFLAGS=-v"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Command().Args = %v, want %v", got, want)
	}
	if got := cmd.Dir; got != "ops" {
		t.Errorf("Command().Dir = %q, want %q", got, "ops")
	}
}

func TestCommandLine(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"plain words pass through", []string{"make", "test"}, "make test"},
		{"path and flag characters pass through", []string{"./bin/tool", "--flag=x,y"}, "./bin/tool --flag=x,y"},
		{"argument with spaces is quoted", []string{"echo", "a b"}, "echo 'a b'"},
		{"single quotes are escaped", []string{"echo", "it's"}, `echo 'it'\''s'`},
		{"empty argument", []string{"cmd", ""}, "cmd ''"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{Source: "rc", Name: "t", Args: tt.args}
			if got := task.CommandLine(); got != tt.want {
				t.Errorf("CommandLine() = %q, want %q", got, tt.want)
			}
		})
	}
}
