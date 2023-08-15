package tracing

import (
	"fmt"
	"os"
	"os/exec"
)

/*
 * /falco/rules.yaml - file that falco uses to get the rules
 * /falco/data.json - traced syscalls list are stored here
 * /falco/logs.txt - falco logs are stored here (this includes the logs from the `formatter` binary)
 */

type TracingConfiguration struct {
	PodName       string            `json:"podName"`
	ContainerName string            `json:"containerName"`
	PodLabel      map[string]string `json:"podLabel"`
}

type Tracer struct {
	falcoProcess *os.Process
	Config       TracingConfiguration
	IsTracing    bool
}

func NewTracer() (Tracer, error) {
	return Tracer{}, nil
}

func (t *Tracer) UpdateConfig(config TracingConfiguration) error {
	t.Config = config
	rule, err := CreateFalcoRule(config)
	if err != nil {
		return err
	}
	// Write the rule to /falco/rules.yaml
	err = os.WriteFile("/falco/rules.yaml", rule, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Start Falco process and update the struct with the falco process
func (t *Tracer) Start() error {
	khost := os.Getenv("KUBERNETES_SERVICE_HOST")
	if khost == "" {
		return fmt.Errorf("KUBERNETES_SERVICE_HOST not set")
	}
	falcoCommand := exec.Command("/usr/bin/falco",
		// to get support for all available syscalls
		"-A",
		// unbuffered makes sure that we don't lose any syscalls
		"-U",
		"-r", "/falco/rules.yaml",
		"-k", "https://"+khost,
		"-K", "/var/run/secrets/kubernetes.io/serviceaccount/token",
		"--option", "program_output.enabled=true",
		"--option", "program_output.keep_alive=true",
		"--option", "program_output.program=/falco/formatter",
		"--option", "stdout_output.enabled=false",
		"--option", "syslog_output.enabled=false",
		"--option", "file_output.enabled=false",
		"--option", "json_output=true",
	)
	falcoCommand.Env = os.Environ()
	falcoCommand.Env = append(falcoCommand.Env, "FALCO_BPF_PROBE=/falco/falco-bpf.o")
	f, _ := os.Create("/falco/logs.txt")
	falcoCommand.Stdout = f
	falcoCommand.Stderr = f
	// we have to call Process.Release when stopping it
	err := falcoCommand.Start()
	if err != nil {
		return err
	}
	t.falcoProcess = falcoCommand.Process
	return nil
}

// Stop the tracer by sending interrupt to the Falco process
// TODO: Make sure that this kills both the processes (falco and formatter)
func (t *Tracer) Stop() error {
	if t.falcoProcess == nil {
		return fmt.Errorf("Process not found in the struct")
	}
	err := (*t.falcoProcess).Signal(os.Interrupt)
	if err != nil {
		return err
	}
	t.falcoProcess.Release()
	t.falcoProcess = nil
	return nil
}
