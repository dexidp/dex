package main

import (
	"log"
	"time"

	"github.com/dexidp/dex/cmd/first-auth/gRPCapi"
	"github.com/spf13/cobra"
)

var genClientTokenCmd = &cobra.Command{
	Use:   "genClientToken",
	Short: "Generate a client token to first auth",
	Long:  "It will generate a client token into the client_token table of dex dB",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to th dex server
		grpcApi, err := ConnectToGrpcApi(cmd)
		if err != nil {
			log.Fatalf("Failed to allocate grpcApi struct: %v", err)
		}
		// Catch data into flags
		if clientID, err := cmd.Flags().GetString("clientID"); clientID != "" {
			if err != nil {
				log.Fatalf("Error occured during getting flag maxUser: %v", err)
			}
			days, err := cmd.Flags().GetInt("days")
			if err != nil {
				log.Fatalf("Error occured during getting flag clientTokens: %v", err)
			}
			clientTokenId := NewID()
			if err := grpcApi.AddClientToken(gRPCapi.ClientToken{Id: clientTokenId, ClientId: clientID, CreatedAt: time.Now(), ExpiredAt: time.Now().AddDate(0, 0, days)}); err != nil {
				log.Fatalf("failed to create ClientToken: %v", err)
			}
			log.Println("A client token has been generated: ", clientTokenId)
		} else {
			log.Fatalf("You forgot to add client ID: ./bin/first-auth genClientTOken --clientID=redmine")
		}

	},
}

func genClientToken() *cobra.Command {
	genClientTokenCmd.Flags().StringP("clientID", "i", "", "Number of maximum utility")
	genClientTokenCmd.Flags().IntP("days", "d", 0, "days to add for expiration ")
	return genClientTokenCmd
}
