package main

import (
	"log"

	"github.com/dexidp/dex/cmd/first-auth/gRPCapi"
	"github.com/spf13/cobra"
)

var genUserIdpCmd = &cobra.Command{
	Use:   "genUserIdp",
	Short: "Generate a userIdp to first auth",
	Long:  "It will generate an UserIdp into the dex dB",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to th dex server
		grpcApi, err := ConnectToGrpcApi(cmd)
		if err != nil {
			log.Fatalf("Failed to allocate grpcApi struct: %v", err)
		}
		// Add new user
		if id, err := cmd.Flags().GetString("userID"); id != "" || err != nil {
			if err != nil {
				log.Fatalf("Error occured during getting flag userID: %v", err)
			}

			// Catch data into flags
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
			internId := NewID()

			if err := grpcApi.AddUserIdp(gRPCapi.UserIdp{IdpId: id, InternId: internId}); err != nil {
				log.Fatalf("failed to create UserIdp: %v", err)
			}
			if err := grpcApi.AddUser(gRPCapi.User{InternId: internId, Pseudo: pseudo, Email: email, AclTokens: aclTokens}); err != nil {
				log.Fatalf("failed to create User: %v", err)
			}
		} else {
			log.Fatalf("You forgot to add user idp ID for the user: ./bin/first-auth genUserIdp --userID=example_ldap")
		}

	},
}

func genUserIdp() *cobra.Command {
	genUserIdpCmd.Flags().StringP("userID", "u", "", "ID of the user idp")
	genUserIdpCmd.Flags().StringP("pseudo", "p", "", "Pseudo of the user")
	genUserIdpCmd.Flags().StringP("email", "e", "", "Email of the user")
	genUserIdpCmd.Flags().StringSlice("aclTokens", []string{}, "list of aclTokens")
	return genUserIdpCmd
}
