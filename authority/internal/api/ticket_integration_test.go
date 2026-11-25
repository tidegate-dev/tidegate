//go:build integration

package api

import (
	"net/http/httptest"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/secretengine/postgres"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/authority/internal/storetest"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/secretengine"
)

var _ = Describe("TicketService", func() {
	var (
		pool       *pgxpool.Pool
		asset      *model.Asset
		alice      model.UserID
		aliceToken secret.String
		client     authorityv1connect.TicketServiceClient
	)

	BeforeEach(func(ctx SpecContext) {
		var err error

		dbPool, data := storetest.New(ctx)
		pool = dbPool
		projects := service.NewProjects(data.Projects(), data)

		Expect(postgres.CreateSchema(ctx, pool)).To(Succeed())

		engines := secretengine.NewEngines(map[string]secretengine.Engine{"test-engine": postgres.New(pool)})
		assets := service.NewAssets(data.Projects(), data.Assets(), data, engines, "test-engine")
		sessions := newSessions(data)
		alice = storetest.AddUser(ctx, data, "alice")
		aliceToken, err = sessions.Issue(ctx, alice)

		Expect(err).NotTo(HaveOccurred())

		_, err = projects.Create(ctx, "payments")

		Expect(err).NotTo(HaveOccurred())

		target := model.SSHTarget{Host: "host.example.com", Port: 22, HostKey: "ssh-ed25519 AAAA", UserName: "deploy"}
		asset, err = assets.CreateSSH(ctx, "payments", "host", nil, target, service.SSHSecretInput{
			Kind:  model.SSHSecretKindPassword,
			Value: secret.NewBytes([]byte("test-password")),
		})

		Expect(err).NotTo(HaveOccurred())

		tickets := service.NewTickets(data)
		_, handler := authorityv1connect.NewTicketServiceHandler(NewTicketService(tickets),
			connect.WithInterceptors(NewSessionInterceptor(sessions), validate.NewInterceptor()))
		server := httptest.NewServer(handler)
		DeferCleanup(server.Close)
		client = authorityv1connect.NewTicketServiceClient(server.Client(), server.URL)
	})

	create := func(ctx SpecContext, assetID string, token secret.String) (*connect.Response[authorityv1.CreateTicketResponse], error) {
		request := connect.NewRequest(&authorityv1.CreateTicketRequest{AssetId: assetID})
		if token.Reveal() != "" {
			request.Header().Set("Authorization", "Bearer "+token.Reveal())
		}

		return client.CreateTicket(ctx, request)
	}

	It("requires a signed-in user", func(ctx SpecContext) {
		_, err := create(ctx, asset.ID.String(), secret.String{})

		Expect(connect.CodeOf(err)).To(Equal(connect.CodeUnauthenticated))
	}, SpecTimeout(30*time.Second))

	It("rejects a malformed asset ID", func(ctx SpecContext) {
		_, err := create(ctx, "invalid", aliceToken)

		Expect(connect.CodeOf(err)).To(Equal(connect.CodeInvalidArgument))
	}, SpecTimeout(30*time.Second))

	It("issues a fresh ticket for the session user and requested asset on every call", func(ctx SpecContext) {
		first, err := create(ctx, asset.ID.String(), aliceToken)

		Expect(err).NotTo(HaveOccurred())
		Expect(first.Msg.GetAssetId()).To(Equal(asset.ID.String()))
		Expect(first.Msg.GetUserId()).To(Equal(alice.String()))
		Expect(first.Msg.GetId()).NotTo(BeEmpty())

		second, err := create(ctx, asset.ID.String(), aliceToken)

		Expect(err).NotTo(HaveOccurred())
		Expect(second.Msg.GetId()).NotTo(BeEmpty())
		Expect(second.Msg.GetId()).NotTo(Equal(first.Msg.GetId()))
	}, SpecTimeout(30*time.Second))

	It("returns not_found for a deleted asset", func(ctx SpecContext) {
		_, err := pool.Exec(ctx, "DELETE FROM assets WHERE id = $1", asset.ID.String())
		Expect(err).NotTo(HaveOccurred())

		_, err = create(ctx, asset.ID.String(), aliceToken)

		Expect(connect.CodeOf(err)).To(Equal(connect.CodeNotFound))
	}, SpecTimeout(30*time.Second))
})
