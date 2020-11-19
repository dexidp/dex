package main

import (
	"log"

	"github.com/spf13/cobra"
)

var delUserIdpCmd = &cobra.Command{
	Use:   "delUserIdp",
	Short: "Delete a userIdp",
	Long:  "It will delete an UserIdp from user_idp table into the dex dB",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to th dex server
		grpcApi, err := ConnectToGrpcApi(cmd)
		if err != nil {
			log.Fatalf("Failed to allocate grpcApi struct: %v", err)
		}
		// del the user
		if id, err := cmd.Flags().GetString("userID"); id != "" || err != nil {
			if err != nil {
				log.Fatalf("Error occured during getting flag userID: %v", err)
			}
			if err := grpcApi.DeleteUserIdp(id); err != nil {
				log.Fatalf("failed to delete user idp: %v", err)
			}
			log.Println("A userIdp has been deleted: ", id)
		} else {
			log.Fatalf("You forgot to add user idp ID for the user: ./bin/first-auth delUserIdp --userID=example_ldap")
		}

	},
}

func delUserIdp() *cobra.Command {
	delUserIdpCmd.Flags().StringP("userID", "u", "", "ID of the user idp")
	return delUserIdpCmd
}
