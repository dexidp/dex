package main

import (
	"log"

	"github.com/dexidp/dex/cmd/first-auth/gRPCapi"
	"github.com/spf13/cobra"
)

var upAclTokenCmd = &cobra.Command{
	Use:   "upAclToken",
	Short: "Update a acl token",
	Long:  "It will update an AclToken into the tabe acl_token of the dex dB",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to th dex server
		grpcApi, err := ConnectToGrpcApi(cmd)
		if err != nil {
			log.Fatalf("Failed to allocate grpcApi struct: %v", err)
		}
		// Catch the ID of the user to update
		if id, err := cmd.Flags().GetString("tokenID"); id != "" {
			if err != nil {
				log.Fatalf("Error occured during getting flag tokenID: %v", err)
			}

			// Catch data into flags
			desc, err := cmd.Flags().GetStringSlice("desc")
			if err != nil {
				log.Fatalf("Error occured during getting flag desc: %v", err)
			}
			description := ""
			for index, descWord := range desc {
				if index == 0 {
					description = descWord
				} else {
					description = description + " " + descWord
				}
			}
			maxUser, err := cmd.Flags().GetString("maxUser")
			if err != nil {
				log.Fatalf("Error occured during getting flag maxUser: %v", err)
			}
			clientTokens, err := cmd.Flags().GetStringSlice("clientTokens")
			if err != nil {
				log.Fatalf("Error occured during getting flag clientTokens: %v", err)
			}

			// Update the corresponding aclToken
			if err := grpcApi.UpdateAclToken(gRPCapi.AclToken{Id: id, Desc: description, MaxUser: maxUser, ClientTokens: clientTokens}); err != nil {
				log.Fatalf("failed to update the acl token: %v", err)
			}
			log.Println("A acl token has been updated: ", id)

		} else {
			log.Fatalf("You forgot to add token ID: ./bin/first-auth upAclToken --tokenID=xxxXXXxxx")
		}

	},
}

func upAclToken() *cobra.Command {
	upAclTokenCmd.Flags().StringP("tokenID", "t", "", "ID of the acl token")
	upAclTokenCmd.Flags().StringSlice("desc", []string{}, "description of this acl token")
	upAclTokenCmd.Flags().StringP("maxUser", "p", "", "Number of maximum utility")
	upAclTokenCmd.Flags().StringSlice("clientTokens", []string{}, "list of client Tokens")
	return upAclTokenCmd
}
