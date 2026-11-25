//go:build integration

package service

import (
	"context"
	"crypto/ed25519"
	"encoding/pem"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"

	"github.com/phemera-dev/phemera/authority/internal/model"
	"github.com/phemera-dev/phemera/authority/internal/sshkey"
	"github.com/phemera-dev/phemera/authority/internal/storetest"
	"github.com/phemera-dev/phemera/internal/secret"
	"github.com/phemera-dev/phemera/internal/secretengine"
)

var _ = Describe("Assets", func() {
	var (
		projects    *Projects
		assets      *Assets
		engine      *fakeEngine
		target      model.SSHTarget
		secretInput SSHSecretInput
	)

	BeforeEach(func(ctx SpecContext) {
		_, data := storetest.New(ctx)
		projects = NewProjects(data.Projects(), data)
		engine = &fakeEngine{values: map[secretengine.Reference][]byte{}}
		engines := secretengine.NewEngines(map[string]secretengine.Engine{"test-engine": engine})
		assets = NewAssets(data.Projects(), data.Assets(), data, engines, "test-engine")
		target = model.SSHTarget{Host: "bastion.example", Port: 22, HostKey: "ssh-ed25519 AAAA", UserName: "deploy"}
		secretInput = SSHSecretInput{
			Kind:  model.SSHSecretKindPassword,
			Value: secret.NewBytes([]byte("the value")),
		}
	})

	Describe("CreateSSH", func() {
		It("returns the new SSH asset of the project, which List then returns too", func(ctx SpecContext) {
			project, err := projects.Create(ctx, "payments")
			Expect(err).NotTo(HaveOccurred())

			asset, err := assets.CreateSSH(ctx, "payments", "bastion", map[string]string{"env": "prod"}, target, secretInput)

			Expect(err).NotTo(HaveOccurred())
			Expect(asset.ID).NotTo(BeZero())
			Expect(asset.ProjectID).To(Equal(project.ID))
			Expect(asset.Name).To(Equal("bastion"))
			Expect(asset.Kind).To(Equal(model.AssetKindSSH))
			Expect(asset.Labels).To(Equal(map[string]string{"env": "prod"}))
			Expect(asset.SSH).To(Equal(&model.SSHTarget{
				Host:     "bastion.example",
				Port:     22,
				HostKey:  "ssh-ed25519 AAAA",
				UserName: "deploy",
				Secret: model.StoredSSHSecret{
					EngineName: "test-engine",
					Reference:  "reference-1",
					Kind:       model.SSHSecretKindPassword,
				},
			}))
			Expect(assets.List(ctx, "payments")).To(ConsistOf(asset))
		})

		It("returns ErrProjectNotFound for a project that doesn't exist and stores nothing in the engine", func(ctx SpecContext) {
			_, err := assets.CreateSSH(ctx, "payments", "bastion", nil, target, secretInput)

			Expect(err).To(MatchError(model.ErrProjectNotFound))
			Expect(engine.values).To(BeEmpty())
		})

		When("the project exists", func() {
			BeforeEach(func(ctx SpecContext) {
				_, err := projects.Create(ctx, "payments")
				Expect(err).NotTo(HaveOccurred())
			})

			It("stores the password in the engine", func(ctx SpecContext) {
				_, err := assets.CreateSSH(ctx, "payments", "bastion", nil, target, secretInput)

				Expect(err).NotTo(HaveOccurred())
				Expect(engine.values).To(Equal(map[secretengine.Reference][]byte{"reference-1": []byte("the value")}))
			})

			It("returns ErrAssetNameTaken when another asset has the name, and deletes nothing from the engine", func(ctx SpecContext) {
				_, err := assets.CreateSSH(ctx, "payments", "bastion", nil, target, secretInput)
				Expect(err).NotTo(HaveOccurred())

				_, err = assets.CreateSSH(ctx, "payments", "bastion", nil, target, secretInput)

				Expect(err).To(MatchError(model.ErrAssetNameTaken))
				Expect(engine.values).To(HaveLen(2))
			})

			It("returns ErrSSHPasswordEmpty for an empty password and stores nothing", func(ctx SpecContext) {
				secretInput.Value = secret.NewBytes(nil)

				_, err := assets.CreateSSH(ctx, "payments", "bastion", nil, target, secretInput)

				Expect(err).To(MatchError(ErrSSHPasswordEmpty))
				Expect(engine.values).To(BeEmpty())
				Expect(assets.List(ctx, "payments")).To(BeEmpty())
			})

			It("stores a private key that has no passphrase in the engine", func(ctx SpecContext) {
				privateKey := newPrivateKey()
				secretInput.Kind = model.SSHSecretKindPrivateKey
				secretInput.Value = secret.NewBytes(privateKey)

				_, err := assets.CreateSSH(ctx, "payments", "bastion", nil, target, secretInput)

				Expect(err).NotTo(HaveOccurred())
				Expect(engine.values).To(Equal(map[secretengine.Reference][]byte{"reference-1": privateKey}))
			})

			It("returns ErrPrivateKeyEncrypted for an encrypted private key and stores nothing", func(ctx SpecContext) {
				secretInput.Kind = model.SSHSecretKindPrivateKey
				secretInput.Value = secret.NewBytes(pem.EncodeToMemory(&pem.Block{
					Type:    "RSA PRIVATE KEY",
					Headers: map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-128-CBC,00000000000000000000000000000000"},
					Bytes:   []byte("the encrypted key"),
				}))

				_, err := assets.CreateSSH(ctx, "payments", "bastion", nil, target, secretInput)

				Expect(err).To(MatchError(sshkey.ErrPrivateKeyEncrypted))
				Expect(engine.values).To(BeEmpty())
				Expect(assets.List(ctx, "payments")).To(BeEmpty())
			})

			It("returns ErrPrivateKeyInvalid for a value that isn't a private key and stores nothing", func(ctx SpecContext) {
				secretInput.Kind = model.SSHSecretKindPrivateKey

				_, err := assets.CreateSSH(ctx, "payments", "bastion", nil, target, secretInput)

				Expect(err).To(MatchError(sshkey.ErrPrivateKeyInvalid))
				Expect(engine.values).To(BeEmpty())
				Expect(assets.List(ctx, "payments")).To(BeEmpty())
			})

			It("stores nothing for a password or private key of an unknown kind", func(ctx SpecContext) {
				secretInput.Kind = "certificate"

				_, err := assets.CreateSSH(ctx, "payments", "bastion", nil, target, secretInput)

				Expect(err).To(MatchError(ContainSubstring(`unknown SSH secret kind "certificate"`)))
				Expect(engine.values).To(BeEmpty())
				Expect(assets.List(ctx, "payments")).To(BeEmpty())
			})
		})
	})

	Describe("List", func() {
		It("returns ErrProjectNotFound for a project that doesn't exist", func(ctx SpecContext) {
			_, err := assets.List(ctx, "payments")

			Expect(err).To(MatchError(model.ErrProjectNotFound))
		})
	})
})

func newPrivateKey() []byte {
	GinkgoHelper()

	_, key, err := ed25519.GenerateKey(nil)
	Expect(err).NotTo(HaveOccurred())

	block, err := ssh.MarshalPrivateKey(key, "")
	Expect(err).NotTo(HaveOccurred())

	return pem.EncodeToMemory(block)
}

type fakeEngine struct {
	values map[secretengine.Reference][]byte
	stored int
}

func (f *fakeEngine) Store(_ context.Context, value secret.Bytes) (secretengine.Reference, error) {
	f.stored++
	reference := secretengine.Reference(fmt.Sprintf("reference-%d", f.stored))
	f.values[reference] = value.Reveal()

	return reference, nil
}

func (f *fakeEngine) Fetch(_ context.Context, reference secretengine.Reference) (secret.Bytes, error) {
	value, ok := f.values[reference]
	if !ok {
		return secret.Bytes{}, secretengine.ErrNotFound
	}

	return secret.NewBytes(value), nil
}

func (f *fakeEngine) Delete(_ context.Context, reference secretengine.Reference) error {
	_, ok := f.values[reference]
	if !ok {
		return secretengine.ErrNotFound
	}

	delete(f.values, reference)

	return nil
}
