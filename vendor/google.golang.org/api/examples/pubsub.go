package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"strings"

	pubsub "google.golang.org/api/pubsub/v1beta1"
)

const USAGE = `Available arguments are:
    PROJ list_topics
    PROJ create_topic TOPIC
    PROJ delete_topic TOPIC
    PROJ list_subscriptions
    PROJ create_subscription SUBSCRIPTION LINKED_TOPIC
    PROJ delete_subscription SUBSCRIPTION
    PROJ connect_irc TOPIC SERVER CHANNEL
    PROJ pull_messages SUBSCRIPTION
`

type IRCBot struct {
	server   string
	port     string
	nick     string
	user     string
	channel  string
	conn     net.Conn
	tpReader *textproto.Reader
}

func NewIRCBot(server, channel, nick string) *IRCBot {
	return &IRCBot{
		server:  server,
		port:    "6667",
		nick:    nick,
		channel: channel,
		conn:    nil,
		user:    nick,
	}
}

func (bot *IRCBot) Connect() {
	conn, err := net.Dial("tcp", bot.server+":"+bot.port)
	if err != nil {
		log.Fatal("unable to connect to IRC server ", err)
	}
	bot.conn = conn
	log.Printf("Connected to IRC server %s (%s)\n",
		bot.server, bot.conn.RemoteAddr())
	bot.tpReader = textproto.NewReader(bufio.NewReader(bot.conn))
	bot.Sendf("USER %s 8 * :%s\r\n", bot.nick, bot.nick)
	bot.Sendf("NICK %s\r\n", bot.nick)
	bot.Sendf("JOIN %s\r\n", bot.channel)
}

func (bot *IRCBot) CheckConnection() {
	for {
		line, err := bot.ReadLine()
		if err != nil {
			log.Fatal("Unable to read a line during checking the connection.")
		}
		if strings.Contains(line, "004") {
			log.Println("The nick accepted.")
		} else if strings.Contains(line, "433") {
			log.Fatal("The nick is already in use.")
		} else if strings.Contains(line, "366") {
			log.Println("Starting to publish messages.")
			return
		}
	}
}

func (bot *IRCBot) Sendf(format string, args ...interface{}) {
	fmt.Fprintf(bot.conn, format, args...)
}

func (bot *IRCBot) Close() {
	bot.conn.Close()
}

func (bot *IRCBot) ReadLine() (line string, err error) {
	return bot.tpReader.ReadLine()
}

func init() {
	registerDemo("pubsub", pubsub.PubsubScope, pubsubMain)
}

func pubsubUsage() {
	fmt.Fprint(os.Stderr, USAGE)
}

// Returns a fully qualified resource name for Cloud Pub/Sub.
func fqrn(res, proj, name string) string {
	return fmt.Sprintf("/%s/%s/%s", res, proj, name)
}

func fullTopicName(proj, topic string) string {
	return fqrn("topics", proj, topic)
}

func fullSubscriptionName(proj, topic string) string {
	return fqrn("subscriptions", proj, topic)
}

// Check the length of the arguments.
func checkArgs(argv []string, min int) {
	if len(argv) < min {
		pubsubUsage()
		os.Exit(2)
	}
}

func listTopics(service *pubsub.Service, argv []string) {
	var nextPageToken string = ""
	for {
		query := service.Topics.List().Query(
			fmt.Sprintf(
				"cloud.googleapis.com/project in (/projects/%s)",
				argv[0])).PageToken(nextPageToken)
		topicsList, err := query.Do()
		if err != nil {
			log.Fatal("Got an error: %v", err)
		}
		for _, topic := range topicsList.Topic {
			fmt.Println(topic.Name)
		}
		nextPageToken = topicsList.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
}

func createTopic(service *pubsub.Service, argv []string) {
	checkArgs(argv, 3)
	topic := &pubsub.Topic{Name: fullTopicName(argv[0], argv[2])}
	topic, err := service.Topics.Create(topic).Do()
	if err != nil {
		log.Fatal("Got an error: %v", err)
	}
	fmt.Printf("Topic %s was created.\n", topic.Name)
}

func deleteTopic(service *pubsub.Service, argv []string) {
	checkArgs(argv, 3)
	topicName := fullTopicName(argv[0], argv[2])
	err := service.Topics.Delete(topicName).Do()
	if err != nil {
		log.Fatal("Got an error: %v", err)
	}
	fmt.Printf("Topic %s was deleted.\n", topicName)
}

func listSubscriptions(service *pubsub.Service, argv []string) {
	var nextPageToken string = ""
	for {
		query := service.Subscriptions.List().Query(
			fmt.Sprintf(
				"cloud.googleapis.com/project in (/projects/%s)",
				argv[0])).PageToken(nextPageToken)
		subscriptionsList, err := query.Do()
		if err != nil {
			log.Fatal("Got an error: %v", err)
		}
		for _, subscription := range subscriptionsList.Subscription {
			sub_text, _ := json.MarshalIndent(subscription, "", "  ")
			fmt.Printf("%s\n", sub_text)
		}
		nextPageToken = subscriptionsList.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
}

func createSubscription(service *pubsub.Service, argv []string) {
	checkArgs(argv, 4)
	subscription := &pubsub.Subscription{
		Name:  fullSubscriptionName(argv[0], argv[2]),
		Topic: fullTopicName(argv[0], argv[3]),
	}
	subscription, err := service.Subscriptions.Create(subscription).Do()
	if err != nil {
		log.Fatal("Got an error: %v", err)
	}
	fmt.Printf("Subscription %s was created.\n", subscription.Name)
}

func deleteSubscription(service *pubsub.Service, argv []string) {
	checkArgs(argv, 3)
	subscriptionName := fullSubscriptionName(argv[0], argv[2])
	err := service.Subscriptions.Delete(subscriptionName).Do()
	if err != nil {
		log.Fatal("Got an error: %v", err)
	}
	fmt.Printf("Subscription %s was deleted.\n", subscriptionName)
}

func connectIRC(service *pubsub.Service, argv []string) {
	checkArgs(argv, 5)
	topicName := fullTopicName(argv[0], argv[2])
	server := argv[3]
	channel := argv[4]
	nick := fmt.Sprintf("bot-%s", argv[2])
	ircbot := NewIRCBot(server, channel, nick)
	ircbot.Connect()
	defer ircbot.Close()
	ircbot.CheckConnection()
	privMark := fmt.Sprintf("PRIVMSG %s :", ircbot.channel)
	for {
		line, err := ircbot.ReadLine()
		if err != nil {
			log.Fatal("Unable to read a line from the connection.")
		}
		parts := strings.Split(line, " ")
		if len(parts) > 0 && parts[0] == "PING" {
			ircbot.Sendf("PONG %s\r\n", parts[1])
		} else {
			pos := strings.Index(line, privMark)
			if pos == -1 {
				continue
			}
			privMsg := line[pos+len(privMark) : len(line)]
			pubsubMessage := &pubsub.PubsubMessage{
				Data: base64.StdEncoding.EncodeToString([]byte(privMsg)),
			}
			publishRequest := &pubsub.PublishRequest{
				Message: pubsubMessage,
				Topic:   topicName,
			}
			service.Topics.Publish(publishRequest).Do()
			log.Println("Published a message to the topic.")
		}
	}
}

func pullMessages(service *pubsub.Service, argv []string) {
	checkArgs(argv, 3)
	subscriptionName := fullSubscriptionName(argv[0], argv[2])
	pullRequest := &pubsub.PullRequest{
		ReturnImmediately: false,
		Subscription:      subscriptionName,
	}
	for {
		pullResponse, err := service.Subscriptions.Pull(pullRequest).Do()
		if err != nil {
			log.Fatal("Got an error while pull a message: %v", err)
		}
		if pullResponse.PubsubEvent.Message != nil {
			data, err := base64.StdEncoding.DecodeString(
				pullResponse.PubsubEvent.Message.Data)
			if err != nil {
				log.Fatal("Got an error while decoding the message: %v", err)
			}
			fmt.Printf("%s\n", data)
			ackRequest := &pubsub.AcknowledgeRequest{
				AckId:        []string{pullResponse.AckId},
				Subscription: subscriptionName,
			}
			service.Subscriptions.Acknowledge(ackRequest).Do()
		}
	}
}

// This example demonstrates calling the Cloud Pub/Sub API. As of 20
// Aug 2014, the Cloud Pub/Sub API is only available if you're
// whitelisted. If you're interested in using it, please apply for the
// Limited Preview program at the following form:
// http://goo.gl/Wql9HL
//
// Also, before running this example, be sure to enable Cloud Pub/Sub
// service on your project in Developer Console at:
// https://console.developers.google.com/
//
// It has 8 subcommands as follows:
//
// PROJ list_topics
// PROJ create_topic TOPIC
// PROJ delete_topic TOPIC
// PROJ list_subscriptions
// PROJ create_subscription SUBSCRIPTION LINKED_TOPIC
// PROJ delete_subscription SUBSCRIPTION
// PROJ connect_irc TOPIC SERVER CHANNEL
// PROJ pull_messages SUBSCRIPTION
//
// You can use either of your alphanumerical or numerial Cloud Project
// ID for PROJ. You can choose any names for TOPIC and SUBSCRIPTION as
// long as they follow the naming rule described at:
// https://developers.google.com/pubsub/overview#names
//
// You can list/create/delete topics/subscriptions by self-explanatory
// subcommands, as well as connect to an IRC channel and publish
// messages from the IRC channel to a specified Cloud Pub/Sub topic by
// the "connect_irc" subcommand, or continuously pull messages from a
// specified Cloud Pub/Sub subscription and display the data by the
// "pull_messages" subcommand.
func pubsubMain(client *http.Client, argv []string) {
	checkArgs(argv, 2)
	service, _ := pubsub.New(client)
	m := map[string]func(service *pubsub.Service, argv []string){
		"list_topics":         listTopics,
		"list_subscriptions":  listSubscriptions,
		"create_topic":        createTopic,
		"delete_topic":        deleteTopic,
		"create_subscription": createSubscription,
		"delete_subscription": deleteSubscription,
		"connect_irc":         connectIRC,
		"pull_messages":       pullMessages,
	}
	f, ok := m[argv[1]]
	if !ok {
		pubsubUsage()
		os.Exit(2)
	}
	f(service, argv)
}
