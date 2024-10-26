package auth

import (
	"bytes"
	"errors"

	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	futils "github.com/gofiber/fiber/v2/utils"
)

func (*AuthService) loadConfigSigners() (openpgp.EntityList, error) {
	return openpgp.ReadArmoredKeyRing(bytes.NewBuffer(futils.UnsafeBytes(utils.SIGNER_PGP_PUBLIC_KEY)))
}

func (m *AuthService) validateConfigSign(payload []byte) (_ []byte, e error) {
	var signblock *clearsign.Block
	if signblock, _ = clearsign.Decode(payload); signblock == nil {
		return nil, errors.New("could not decode PGP signed file, clear sign not found")
	}

	var signer *openpgp.Entity
	if signer, e = signblock.VerifySignature(m.signers, m.pgpconfig); e != nil {
		return
	}

	for _, identity := range signer.Identities {
		m.log.Info().Msgf("found trusted identity %s", identity.Name)
	}

	m.log.Info().Msg("received payload has been verified and approved")
	return signblock.Bytes, e
}
