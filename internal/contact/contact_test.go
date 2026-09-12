package contact

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

type sesClientStub struct {
	input *sesv2.SendEmailInput
}

func (s *sesClientStub) SendEmail(_ context.Context, input *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	s.input = input
	return &sesv2.SendEmailOutput{}, nil
}

func TestSESSenderUsesTheContactMailboxAndReplyToAddress(t *testing.T) {
	client := &sesClientStub{}
	sender := NewSESSender(client, "hello@inspectmesh.com")

	err := sender.Send(context.Background(), Message{
		Reason: "General question",
		Email:  "ada@example.com",
		Body:   "Hello",
	})

	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if aws.ToString(client.input.FromEmailAddress) != "hello@inspectmesh.com" || client.input.Destination.ToAddresses[0] != "hello@inspectmesh.com" {
		t.Errorf("email destination = %+v", client.input)
	}
	if client.input.ReplyToAddresses[0] != "ada@example.com" {
		t.Errorf("reply-to = %q", client.input.ReplyToAddresses[0])
	}
}

func TestSESSenderNotifiesTheContactMailboxAboutNewSubscribers(t *testing.T) {
	client := &sesClientStub{}
	sender := NewSESSender(client, "hello@inspectmesh.com")

	err := sender.SendSubscription(context.Background(), "ada@example.com")

	if err != nil {
		t.Fatalf("SendSubscription() error = %v", err)
	}
	if got := aws.ToString(client.input.Content.Simple.Subject.Data); got != "New subscriber: ada@example.com" {
		t.Errorf("subject = %q", got)
	}
	if got := client.input.Destination.ToAddresses[0]; got != "hello@inspectmesh.com" {
		t.Errorf("destination = %q", got)
	}
}
