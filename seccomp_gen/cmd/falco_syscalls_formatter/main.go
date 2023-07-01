package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
)

/*
  This binary takes the falco syscalls stream as the input and formats them and writes them to a file
*/

type falcoOutput struct {
  Hostname string `json:"hostname"`
  Output string `json:"output"`
  Priority string `json:"priority"`
  Rule string `json:"rule"`
  Source string `json:"source"`
  Time string `json:"time"`
  OutputFields map[string]interface{} `json:"output_fields"`
}


func main() {
  scanner := bufio.NewScanner(os.Stdin)
  syscallsMap := make(map[string]struct{})
  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt)
  go func(c chan os.Signal){
    <-c
    fmt.Println("Caught SIGINT, exiting...")
    // write the syscalls to a data.json file
    syscallData := make([]string, 0)
    for syscall := range syscallsMap {
      syscallData = append(syscallData, syscall)
    }
    file, _ := json.Marshal(syscallData)
    _ = os.WriteFile("/falco/data.json", file, 0644)
    os.Exit(0)
  }(c)
  for scanner.Scan() {
        val := falcoOutput{}
        text := scanner.Text()
        err := json.Unmarshal([]byte(text), &val)
        if err != nil {
          fmt.Println("Error : ", err)
        }
        syscall, ok :=  val.OutputFields["syscall.type"].(string)
        if ok && syscall != "" {
          syscallsMap[syscall] = struct{}{}
        }
  }
  if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading from stdin:", err)
	}
}

