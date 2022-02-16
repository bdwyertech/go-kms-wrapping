package aead

import (
	"crypto/sha256"
	"fmt"
	"io"

	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-kms-wrapping/v2/multi"
	"golang.org/x/crypto/hkdf"
)

// DerivedReader returns a reader from which keys can be read, using the
// given wrapper, reader length limit, salt and context info. Salt and info can
// be nil.
//
// Example:
//	reader, _ := NewDerivedReader(wrapper, userId, jobId)
// 	key := ed25519.GenerateKey(reader)
func NewDerivedReader(wrapper wrapping.Wrapper, lenLimit int64, salt, info []byte) (*io.LimitedReader, error) {
	const (
		op     = "reader.NewDerivedReader"
		minLen = 20
	)
	if wrapper == nil {
		return nil, fmt.Errorf("%s: missing wrapper: %w", op, ErrInvalidParameter)
	}
	if lenLimit < minLen {
		return nil, fmt.Errorf("%s: lenLimit must be >= %d: %w", op, minLen, ErrInvalidParameter)
	}
	type byter interface {
		GetKeyBytes() []byte
	}
	var b byter
	switch w := wrapper.(type) {
	case *multi.PooledWrapper:
		raw := w.WrapperForKeyId("__base__")
		var ok bool
		if b, ok = raw.(byter); !ok {
			return nil, fmt.Errorf("%s: unexpected wrapper type from multiwrapper base: %w", op, ErrInvalidParameter)
		}
	case *Wrapper:
		if w.GetKeyBytes() == nil {
			return nil, fmt.Errorf("%s: aead wrapper missing bytes: %w", op, ErrInvalidParameter)
		}
		b = w
	case *wrapping.TestWrapper:
		if w.GetKeyBytes() == nil {
			return nil, fmt.Errorf("%s: test wrapper missing bytes: %w", op, ErrInvalidParameter)
		}
		b = w
	default:
		return nil, fmt.Errorf("%s: unknown wrapper type: %w", op, ErrInvalidParameter)
	}
	reader := hkdf.New(sha256.New, b.GetKeyBytes(), salt, info)
	return &io.LimitedReader{
		R: reader,
		N: lenLimit,
	}, nil
}
