package main

import (
	"os"

	"github.com/apprenda/kismatic-provision/provision/aws"
	"github.com/apprenda/kismatic-provision/provision/packet"
	"github.com/apprenda/kismatic-provision/provision/vagrant"
	"github.com/sashajeltuhin/ket/provision/openstack"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision is a tool for making Kubernetes capable infrastructure",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	SilenceUsage: true,
}

func init() {

	rootCmd.AddCommand(aws.Cmd())
	rootCmd.AddCommand(packet.Cmd())
	rootCmd.AddCommand(vagrant.Cmd())
	rootCmd.AddCommand(openstack.Cmd())

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
