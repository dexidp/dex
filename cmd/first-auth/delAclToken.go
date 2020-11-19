package main

import (
	"log"

	"github.com/spf13/cobra"
)

var delAclTokenCmd = &cobra.Command{
	Use:   "delAclToken",
	Short: "Delete a acl token",
	Long:  "It will delete an AclToken from acl_token table into the dex dB",
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
			if err := grpcApi.DeleteAclToken(id); err != nil {
				log.Fatalf("failed to delete acl token: %v", err)
			}
			log.Println("A acl token has been deleted: ", id)
		} else {
			log.Fatalf("You forgot to add acl token ID for the token: ./bin/first-auth delAclToken --tokenID=xxxXXXxxx")
		}

	},
}

func delAclToken() *cobra.Command {
	delAclTokenCmd.Flags().StringP("tokenID", "t", "", "ID of the acl token")
	return delAclTokenCmd
}
