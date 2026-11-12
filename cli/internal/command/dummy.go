package command

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

type dummyWriteResult struct {
	ID int64 `json:"id" yaml:"id"`
}

func (result dummyWriteResult) String() string { return strconv.FormatInt(result.ID, 10) }

type dummyReadResult struct {
	Text string `json:"text" yaml:"text"`
}

func (result dummyReadResult) String() string { return result.Text }

func dummyCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:  "dummy",
		Usage: "store texts in the authority's database and read them back",
		Commands: []*cli.Command{
			{
				Name:      "write",
				Usage:     "store a text and print its ID",
				ArgsUsage: "<text>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return dummyWrite(ctx, cmd, store)
				},
			},
			{
				Name:      "read",
				Usage:     "print the text with the given ID",
				ArgsUsage: "<id>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return dummyRead(ctx, cmd, store)
				},
			},
		},
	}
}

func dummyWrite(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 1 {
		return errors.New("want exactly one text")
	}

	text := cmd.Args().First()

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Dummy().Write(ctx, connect.NewRequest(&authorityv1.WriteRequest{Text: text}))
	if err != nil {
		return callFailed("storing the text", connection, err)
	}

	return printResult(cmd, dummyWriteResult{ID: response.Msg.GetId()})
}

func dummyRead(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 1 {
		return errors.New("want exactly one ID")
	}

	id, err := strconv.ParseInt(cmd.Args().First(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID %q, want a number", cmd.Args().First())
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Dummy().Read(ctx, connect.NewRequest(&authorityv1.ReadRequest{Id: id}))
	if err != nil {
		return callFailed("reading the text", connection, err)
	}

	return printResult(cmd, dummyReadResult{Text: response.Msg.GetText()})
}
