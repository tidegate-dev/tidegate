//go:build integration

package api

import (
	"net/http/httptest"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/authority/internal/storetest"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

var _ = Describe("TicketService", func() {
	It("issues a fresh ticket per request while the binding and role permission check assumes access", func(ctx SpecContext) {
		_, data := storetest.New(ctx)
		projects := service.NewProjects(data.Projects(), data)
		assets := service.NewAssets(data.Projects(), data.Assets(), data)
		sessions := newSessions(data)
		alice := storetest.AddUser(ctx, data, "alice")
		aliceToken, err := sessions.Issue(ctx, alice)
		Expect(err).NotTo(HaveOccurred())
		_, err = projects.Create(ctx, "payments")
		Expect(err).NotTo(HaveOccurred())

		target := model.SSHTarget{Host: "host.example.com", Port: 22, HostKey: "ssh-ed25519 AAAA"}
		asset, err := assets.CreateSSH(ctx, "payments", "host", nil, target)
		Expect(err).NotTo(HaveOccurred())

		tickets := service.NewTickets(data)
		_, handler := authorityv1connect.NewTicketServiceHandler(NewTicketService(tickets),
			connect.WithInterceptors(NewSessionInterceptor(sessions), validate.NewInterceptor()))
		server := httptest.NewServer(handler)
		DeferCleanup(server.Close)
		client := authorityv1connect.NewTicketServiceClient(server.Client(), server.URL)
		create := func(assetID string, token secret.String) (*connect.Response[authorityv1.CreateTicketResponse], error) {
			request := connect.NewRequest(&authorityv1.CreateTicketRequest{AssetId: assetID})
			if token.Reveal() != "" {
				request.Header().Set("Authorization", "Bearer "+token.Reveal())
			}

			return client.CreateTicket(ctx, request)
		}

		_, err = create(asset.ID.String(), secret.String{})
		Expect(connect.CodeOf(err)).To(Equal(connect.CodeUnauthenticated))
		_, err = create("invalid", aliceToken)
		Expect(connect.CodeOf(err)).To(Equal(connect.CodeInvalidArgument))
		response, err := create(asset.ID.String(), aliceToken)
		Expect(err).NotTo(HaveOccurred())
		Expect(response.Msg.GetAssetId()).To(Equal(asset.ID.String()))
		Expect(response.Msg.GetUserId()).To(Equal(alice.String()))

		Expect(response.Msg.GetId()).NotTo(BeEmpty())

		again, err := create(asset.ID.String(), aliceToken)
		Expect(err).NotTo(HaveOccurred())
		Expect(again.Msg.GetId()).NotTo(Equal(response.Msg.GetId()))
	}, SpecTimeout(30*time.Second))
})
