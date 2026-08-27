package link

import (
	"crypto/rand"
	"math/big"
)

const slugAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const slugLength = 7

// GenerateSlug returns a random base62 slug. Collisions are handled by the
// caller retrying on ErrSlugTaken, since links.slug is a unique DB constraint.
func GenerateSlug() (string, error) {
	b := make([]byte, slugLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(slugAlphabet))))
		if err != nil {
			return "", err
		}
		b[i] = slugAlphabet[n.Int64()]
	}
	return string(b), nil
}
