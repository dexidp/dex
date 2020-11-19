package main

import (
	"log"

	"github.com/spf13/cobra"
)

var delUserCmd = &cobra.Command{
	Use:   "delUser",
	Short: "Delete a User",
	Long:  "It will delete an User from user table into the dex dB",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to th dex server
		grpcApi, err := ConnectToGrpcApi(cmd)
		if err != nil {
			log.Fatalf("Failed to allocate grpcApi struct: %v", err)
		}
		// del the user
		if id, err := cmd.Flags().GetString("internID"); id != "" || err != nil {
			if err != nil {
				log.Fatalf("Error occured during getting flag userID: %v", err)
			}
			if err := grpcApi.DeleteUser(id); err != nil {
				log.Fatalf("failed to delete user : %v", err)
			}
			log.Println("A User has been deleted: ", id)
		} else {
			log.Fatalf("You forgot to add user ID for the user: ./bin/first-auth delUser --userID=xxxx")
		}

	},
}

func delUser() *cobra.Command {
	delUserCmd.Flags().StringP("internID", "i", "", "ID of the user")
	return delUserCmd
}
