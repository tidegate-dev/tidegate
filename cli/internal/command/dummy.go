package command

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

type dummyWriteResult struct {
	ID int64 `json:"id" yaml:"id"`
}

func (r dummyWriteResult) String() string { return strconv.FormatInt(r.ID, 10) }

type dummyReadResult struct {
	Text string `json:"text" yaml:"text"`
}

func (r dummyReadResult) String() string { return r.Text }

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

	clients, err := dialAuthority(cmd)
	if err != nil {
		return err
	}

	response, err := clients.Dummy().Write(ctx, connect.NewRequest(&authorityv1.WriteRequest{Text: cmd.Args().First()}))
	if err != nil {
		return fmt.Errorf("storing the text at %s: %w", server, err)
	}

	return printResult(cmd, dummyWriteResult{ID: response.Msg.GetId()})
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

	clients, err := dialAuthority(cmd)
	if err != nil {
		return err
	}

	response, err := clients.Dummy().Read(ctx, connect.NewRequest(&authorityv1.ReadRequest{Id: id}))
	if err != nil {
		return fmt.Errorf("reading the text at %s: %w", server, err)
	}

	return printResult(cmd, dummyReadResult{Text: response.Msg.GetText()})
}
