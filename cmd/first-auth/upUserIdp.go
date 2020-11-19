package main

import (
	"log"

	"github.com/dexidp/dex/cmd/first-auth/gRPCapi"
	"github.com/spf13/cobra"
)

var upUserIdpCmd = &cobra.Command{
	Use:   "upUserIdp",
	Short: "Update a userIdp",
	Long:  "It will update an UserIdp into the tabe user_idp of the dex dB",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to th dex server
		grpcApi, err := ConnectToGrpcApi(cmd)
		if err != nil {
			log.Fatalf("Failed to allocate grpcApi struct: %v", err)
		}
		// Catch the ID of the user to update
		if id, err := cmd.Flags().GetString("userID"); id != "" || err != nil {
			if err != nil {
				log.Fatalf("Error occured during getting flag userID: %v", err)
			}

			// Catch data into flags
			internID, err := cmd.Flags().GetString("internID")
			if err != nil {
				log.Fatalf("Error occured during getting flag internID: %v", err)
			}
			pseudo, err := cmd.Flags().GetString("pseudo")
			if err != nil {
				log.Fatalf("Error occured during getting flag pseudo: %v", err)
			}
			email, err := cmd.Flags().GetString("email")
			if err != nil {
				log.Fatalf("Error occured during getting flag email: %v", err)
			}
			aclTokens, err := cmd.Flags().GetStringSlice("aclTokens")
			if err != nil {
				log.Fatalf("Error occured during getting flag email: %v", err)
			}

			// Catch the internal ID
			user, err := grpcApi.GetUserIdp(id)
			if err != nil {
				log.Fatalf("failed to getting an user idp: %v", err)
			}
			if internID != "" {
				if err := grpcApi.UpdateUserIdp(gRPCapi.UserIdp{IdpId: id, InternId: internID}); err != nil {
					log.Fatalf("failed to update the user idp: %v", err)
				}
				user.InternId = internID
			}
			if user.InternId != "" {
				// Update the corresponding user
				if err := grpcApi.UpdateUser(gRPCapi.User{InternId: user.InternId, Pseudo: pseudo, Email: email, AclTokens: aclTokens}); err != nil {
					log.Fatalf("failed to update the user: %v", err)
				}
				log.Println("A user has been updated: ", user.InternId)
			}
		} else {
			log.Fatalf("You forgot to add user idp ID for the user: ./bin/first-auth genUserIdp --userID=example_ldap")
		}

	},
}

func upUserIdp() *cobra.Command {
	upUserIdpCmd.Flags().StringP("internID", "i", "", "intern ID of the user idp")
	upUserIdpCmd.Flags().StringP("userID", "u", "", "ID of the user idp")
	upUserIdpCmd.Flags().StringP("pseudo", "p", "", "user pseudo")
	upUserIdpCmd.Flags().StringP("email", "e", "", "ID of the user idp")
	upUserIdpCmd.Flags().StringSlice("aclTokens", []string{}, "list of aclTokens")
	return upUserIdpCmd
}
