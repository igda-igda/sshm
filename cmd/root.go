package cmd

import (
  "fmt"
  "os"

  "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
  Use:   "sshm",
  Short: "SSH Connection Manager with tmux integration",
  Long: `SSHM is a CLI SSH connection manager that helps DevOps engineers, 
system administrators, and developers connect to multiple remote servers 
simultaneously through organized tmux sessions.`,
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}

func init() {
  rootCmd.AddCommand(addCmd)
  rootCmd.AddCommand(listCmd)
  rootCmd.AddCommand(removeCmd)
  rootCmd.AddCommand(connectCmd)
}