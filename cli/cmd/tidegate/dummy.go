package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
)

func dummyCommand() *cli.Command {
	return &cli.Command{
		Name:  "dummy",
		Usage: "store texts in the authority's database and read them back",
		Commands: []*cli.Command{
			{
				Name:      "write",
				Usage:     "store a text and print its ID",
				ArgsUsage: "<text>",
				Action:    dummyWrite,
			},
			{
				Name:      "read",
				Usage:     "print the text with the given ID",
				ArgsUsage: "<id>",
				Action:    dummyRead,
			},
		},
	}
}

func dummyWrite(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return errors.New("want exactly one text")
	}

	server := cmd.String("server")

	err := checkServer(server)
	if err != nil {
		return err
	}

	client := authorityv1connect.NewDummyServiceClient(http.DefaultClient, server)

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	request := connect.NewRequest(&authorityv1.WriteRequest{Text: cmd.Args().First()})
	setToken(request.Header())

	response, err := client.Write(ctx, request)
	if err != nil {
		return fmt.Errorf("storing the text at %s: %w", server, err)
	}

	_, err = fmt.Fprintln(cmd.Root().Writer, response.Msg.GetId())

	return err
}

func dummyRead(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return errors.New("want exactly one ID")
	}

	id, err := strconv.ParseInt(cmd.Args().First(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID %q, want a number", cmd.Args().First())
	}

	server := cmd.String("server")

	err = checkServer(server)
	if err != nil {
		return err
	}

	client := authorityv1connect.NewDummyServiceClient(http.DefaultClient, server)

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	request := connect.NewRequest(&authorityv1.ReadRequest{Id: id})
	setToken(request.Header())

	response, err := client.Read(ctx, request)
	if err != nil {
		return fmt.Errorf("reading the text at %s: %w", server, err)
	}

	_, err = fmt.Fprintln(cmd.Root().Writer, response.Msg.GetText())

	return err
}

func checkServer(server string) error {
	parsed, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("invalid --server %q: %w", server, err)
	}

	if parsed.Scheme != "http" {
		return nil
	}

	switch parsed.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return nil
	default:
		return fmt.Errorf("refusing to send a password or a token over plain HTTP to %s, use https:// or a server on localhost", server)
	}
}

func setToken(header http.Header) {
	token := os.Getenv("TIDEGATE_TOKEN")
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
}
