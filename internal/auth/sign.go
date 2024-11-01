package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
		m.log.Info().Msgf("found trusted identity %s in received payload", identity.Name)
	}

	m.log.Info().Msg("received payload has been verified and approved")
	return signblock.Bytes, e
}

func (*AuthService) PrepareHMACMessage(size int, payload ...string) []byte {
	payloadlen := len(payload)
	if size == 0 || payloadlen == 0 {
		return nil
	}

	message := make([]byte, 0, size)
	for i, chunk := range payload {
		message = append(message, futils.UnsafeBytes(chunk)...)

		if i+1 <= payloadlen {
			message = append(message, byte(':'))
		}
	}

	return message
}

func (m *AuthService) VerifyHMACSign(message, signed []byte) (string, bool) {
	var buf256 [sha256.Size]byte

	buf, elen :=
		bytes.NewBuffer(buf256[:]),
		hex.EncodedLen(sha256.Size)
	mac := hmac.New(sha256.New, futils.UnsafeBytes(m.token))

	buf.Reset()
	buf.Grow(elen)
	for i := 0; i < elen; i++ {
		buf.WriteByte(0)
	}

	if _, e := mac.Write(message); e != nil {
		m.log.Error().Msg("an error occurred while writing in hmac buffer, " + e.Error())
		return "", false
	}

	// todo save all buffers for reusing
	hex.Encode(buf.Bytes(), mac.Sum(buf256[:0]))
	return buf.String(), bytes.Equal(buf.Bytes(), signed)
}
