package main

import (
	"log"

	"github.com/dexidp/dex/cmd/first-auth/gRPCapi"
	"github.com/spf13/cobra"
)

var genAclTokenCmd = &cobra.Command{
	Use:   "genAclToken",
	Short: "Generate an acl token to first auth",
	Long:  "It will generate an Acl token into the acl_token table of dex dB",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to th dex server
		grpcApi, err := ConnectToGrpcApi(cmd)
		if err != nil {
			log.Fatalf("Failed to allocate grpcApi struct: %v", err)
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
		aclTokenId := NewID()
		if err := grpcApi.AddAclToken(gRPCapi.AclToken{Id: aclTokenId, Desc: description, MaxUser: maxUser, ClientTokens: clientTokens}); err != nil {
			log.Fatalf("failed to create AclToken: %v", err)
		}
		log.Println("A acl token has been generated: ", aclTokenId)

	},
}

func genAclToken() *cobra.Command {
	genAclTokenCmd.Flags().StringSlice("desc", []string{}, "description of this acl token")
	genAclTokenCmd.Flags().StringP("maxUser", "p", "", "Number of maximum utility")
	genAclTokenCmd.Flags().StringSlice("clientTokens", []string{}, "list of client Tokens")
	return genAclTokenCmd
}
