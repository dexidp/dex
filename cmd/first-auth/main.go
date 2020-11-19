package main

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/dexidp/dex/cmd/first-auth/gRPCapi"
	"github.com/spf13/cobra"
)

var encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567")

func commandRoot() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "first-auth",
		Short: "Dex client will interact with first auth on dex server",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(2)
		},
	}

	rootCmd.AddCommand(genUserIdp())
	rootCmd.AddCommand(upUserIdp())
	rootCmd.AddCommand(delUserIdp())

	rootCmd.AddCommand(delUser())

	rootCmd.AddCommand(genAclToken())
	rootCmd.AddCommand(upAclToken())
	rootCmd.AddCommand(delAclToken())

	rootCmd.AddCommand(genClientToken())
	rootCmd.AddCommand(upClientToken())
	rootCmd.AddCommand(delClientToken())

	rootCmd.PersistentFlags().StringP("ca-crt", "a", "/home/devel/workspace/src/dex/ca.crt", "CA certificate")
	rootCmd.PersistentFlags().StringP("client-crt", "c", "/home/devel/workspace/src/dex/client.crt", "Client certificate")
	rootCmd.PersistentFlags().StringP("client-key", "k", "/home/devel/workspace/src/dex/client.key", "Client key")
	return rootCmd
}

func main() {
	fmt.Println("Starting Dex CLI for first authentification with Redpesk patches")
	if err := commandRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
}

func ConnectToGrpcApi(cmd *cobra.Command) (*gRPCapi.GrpcApiDex, error) {
	// Catch flags
	caCrt, _ := cmd.Flags().GetString("ca-crt")
	clientCrt, _ := cmd.Flags().GetString("client-crt")
	clientKey, _ := cmd.Flags().GetString("client-key")
	// Catch Ip adress
	ip := ""
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		os.Stderr.WriteString("Oops: " + err.Error() + "\n")
		os.Exit(1)
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
			}
		}
	}
	// Connect to grpc api of Dex
	grpcApi, err := gRPCapi.NewGrpcApiDex(ip+":5557", caCrt, clientCrt, clientKey)
	if err != nil {
		return nil, err
	}
	return grpcApi, nil
}

func NewID() string {
	return newSecureID(16)
}

func newSecureID(len int) string {
	buff := make([]byte, len) // random ID.
	if _, err := io.ReadFull(rand.Reader, buff); err != nil {
		panic(err)
	}
	// Avoid the identifier to begin with number and trim padding
	return string(buff[0]%26+'a') + strings.TrimRight(encoding.EncodeToString(buff[1:]), "=")
}
