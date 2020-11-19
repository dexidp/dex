package main

import (
	"log"

	"github.com/spf13/cobra"
)

var delClientTokenCmd = &cobra.Command{
	Use:   "delClientToken",
	Short: "Delete a client token",
	Long:  "It will delete an ClientToken from client_token table into the dex dB",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to th dex server
		grpcApi, err := ConnectToGrpcApi(cmd)
		if err != nil {
			log.Fatalf("Failed to allocate grpcApi struct: %v", err)
		}
		// del the token
		if id, err := cmd.Flags().GetString("tokenID"); id != "" || err != nil {
			if err != nil {
				log.Fatalf("Error occured during getting flag tokenID: %v", err)
			}
			if err := grpcApi.DeleteClientToken(id); err != nil {
				log.Fatalf("failed to delete client token: %v", err)
			}
			log.Println("A client token has been deleted: ", id)
		} else {
			log.Fatalf("You forgot to add client token ID for the token: ./bin/first-auth delClientToken --tokenID=xxxXXXxxx")
		}

	},
}

func delClientToken() *cobra.Command {
	delClientTokenCmd.Flags().StringP("tokenID", "t", "", "ID of the acl token")
	return delClientTokenCmd
}
