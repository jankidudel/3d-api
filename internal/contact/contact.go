package contact

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

func IsMessageRejected(err error) bool {
	var rejected *types.MessageRejected
	return errors.As(err, &rejected)
}

type Message struct {
	Reason  string
	Name    string
	Email   string
	Body    string
	Context string
}

type Sender interface {
	Send(context.Context, Message) error
}

type SubscriberSender interface {
	SendSubscription(context.Context, string) error
}

type SESClient interface {
	SendEmail(context.Context, *sesv2.SendEmailInput, ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

type SESSender struct {
	client      SESClient
	destination string
}

func NewSESSender(client SESClient, destination string) *SESSender {
	return &SESSender{client: client, destination: destination}
}

func (s *SESSender) Send(ctx context.Context, message Message) error {
	_, err := s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.destination),
		Destination:      &types.Destination{ToAddresses: []string{s.destination}},
		ReplyToAddresses: []string{message.Email},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(fmt.Sprintf("InspectMesh feedback: %s", message.Reason))},
			Body: &types.Body{Text: &types.Content{Data: aws.String(fmt.Sprintf(
				"Reason: %s\nName: %s\nEmail: %s\nPage or tool: %s\n\n%s",
				message.Reason,
				valueOrDefault(message.Name, "Not provided"),
				message.Email,
				valueOrDefault(message.Context, "Not provided"),
				message.Body,
			))}},
		}},
	})
	return err
}

func (s *SESSender) SendSubscription(ctx context.Context, email string) error {
	_, err := s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.destination),
		Destination:      &types.Destination{ToAddresses: []string{s.destination}},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String("New subscriber: " + email)},
			Body:    &types.Body{Text: &types.Content{Data: aws.String("New InspectMesh subscriber: " + email)}},
		}},
	})
	return err
}

func valueOrDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
