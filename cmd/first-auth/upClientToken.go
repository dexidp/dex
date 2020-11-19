package main

import (
	"log"
	"time"

	"github.com/dexidp/dex/cmd/first-auth/gRPCapi"
	"github.com/spf13/cobra"
)

var upClientTokenCmd = &cobra.Command{
	Use:   "upClientToken",
	Short: "Update a client token",
	Long:  "It will update an ClientToken into the tabe client_token of the dex dB",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to th dex server
		grpcApi, err := ConnectToGrpcApi(cmd)
		if err != nil {
			log.Fatalf("Failed to allocate grpcApi struct: %v", err)
		}
		// Catch the ID of the user to update
		if id, err := cmd.Flags().GetString("tokenID"); id != "" || err != nil {
			if err != nil {
				log.Fatalf("Error occured during getting flag tokenID: %v", err)
			}

			// Catch data into flags
			clientID, err := cmd.Flags().GetString("clientID")
			if err != nil {
				log.Fatalf("Error occured during getting flag clientID: %v", err)
			}
			days, err := cmd.Flags().GetInt64("days")
			if err != nil {
				log.Fatalf("Error occured during getting flag days: %v", err)
			}

			// Catch older value
			token, err := grpcApi.GetClientToken(id)
			if err != nil {
				log.Fatalf("failed to get client token: %v", err)
			}
			// add days to the expired data
			date := time.Unix(token.ExpiredAt, 0)
			date = date.AddDate(0, 0, int(days))
			if date.Unix() < time.Now().Unix() {
				date = time.Now()
			}

			// Update the corresponding aclToken
			if err := grpcApi.UpdateClientToken(gRPCapi.ClientToken{Id: id, ClientId: clientID, CreatedAt: time.Unix(token.CreatedAt, 0), ExpiredAt: date}); err != nil {
				log.Fatalf("failed to update the client token: %v", err)
			}
			log.Println("A client token has been updated: ", id)

		} else {
			log.Fatalf("You forgot to add token ID: ./bin/first-auth upClientToken --tokenID=xxxXXXxxx")
		}

	},
}

func upClientToken() *cobra.Command {
	upClientTokenCmd.Flags().StringP("tokenID", "t", "", "ID of the client token")
	upClientTokenCmd.Flags().StringP("clientID", "i", "", "description of the token")
	upClientTokenCmd.Flags().Int64P("days", "d", 0, "numbers of days to add / delete for the expired time")
	return upClientTokenCmd
}
