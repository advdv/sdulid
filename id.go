// Package sdulid implements an kind of ulid that self-describes which entity it represents.
//
//nolint:mnd
package sdulid

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/oklog/ulid/v2"
)

var (
	// ErrNoPrefix is returned when a self-describing ulid is parsed without a prefix.
	ErrNoPrefix = errors.New("sdulid: no prefix")
	// ErrInvalidSuffix is returned during text decoding when the long form (no prefix) ulid is
	// provided and the last two bytes don't match what is expected for the type that it's decoding into.
	ErrInvalidSuffix = errors.New("sdulid: invalid ulid suffix")
	// ErrBufferSize is returned when marshalling ULIDs to a buffer of insufficient size.
	ErrBufferSize = errors.New("sdulid: bad buffer size when marshaling")
)

// ID type used for all entity IDs.
type ID[T Kind] struct{ ulid.ULID }

func (id *ID[T]) putSuffixBytes() {
	var kind T
	binary.BigEndian.PutUint16(id.ULID[14:], kind.KindNumber())
}

func (id ID[T]) String() string {
	d, _ := id.MarshalText()

	return string(d)
}

// PrefixSize returns the size of the prefix for text encoding.
func (id ID[T]) PrefixSize() int {
	var kind T

	return len(kind.KindShortIdent()) + 1
}

// EncodedSize return the size of a text-encoded self-describing ulid.
func (id ID[T]) EncodedSize() int {
	return id.PrefixSize() + ulid.EncodedSize - binary.Size(uint16(0))
}

// MarshalTextTo encodes the id in its text representation with the prefix.
func (id ID[T]) MarshalTextTo(dst []byte) error {
	if len(dst) != id.EncodedSize() {
		return ErrBufferSize
	}

	// write the prefix to the buffer.
	var kind T
	copy(dst, []byte(kind.KindShortIdent()+"_"))

	// Optimized unrolled loop ahead.
	// From https://github.com/RobThree/NUlid
	plen := id.PrefixSize()
	// 10 byte timestamp
	dst[plen+0] = ulid.Encoding[(id.ULID[0]&224)>>5]
	dst[plen+1] = ulid.Encoding[id.ULID[0]&31]
	dst[plen+2] = ulid.Encoding[(id.ULID[1]&248)>>3]
	dst[plen+3] = ulid.Encoding[((id.ULID[1]&7)<<2)|((id.ULID[2]&192)>>6)]
	dst[plen+4] = ulid.Encoding[(id.ULID[2]&62)>>1]
	dst[plen+5] = ulid.Encoding[((id.ULID[2]&1)<<4)|((id.ULID[3]&240)>>4)]
	dst[plen+6] = ulid.Encoding[((id.ULID[3]&15)<<1)|((id.ULID[4]&128)>>7)]
	dst[plen+7] = ulid.Encoding[(id.ULID[4]&124)>>2]
	dst[plen+8] = ulid.Encoding[((id.ULID[4]&3)<<3)|((id.ULID[5]&224)>>5)]
	dst[plen+9] = ulid.Encoding[id.ULID[5]&31]

	// 16 bytes of entropy
	dst[plen+10] = ulid.Encoding[(id.ULID[6]&248)>>3]
	dst[plen+11] = ulid.Encoding[((id.ULID[6]&7)<<2)|((id.ULID[7]&192)>>6)]
	dst[plen+12] = ulid.Encoding[(id.ULID[7]&62)>>1]
	dst[plen+13] = ulid.Encoding[((id.ULID[7]&1)<<4)|((id.ULID[8]&240)>>4)]
	dst[plen+14] = ulid.Encoding[((id.ULID[8]&15)<<1)|((id.ULID[9]&128)>>7)]
	dst[plen+15] = ulid.Encoding[(id.ULID[9]&124)>>2]
	dst[plen+16] = ulid.Encoding[((id.ULID[9]&3)<<3)|((id.ULID[10]&224)>>5)]
	dst[plen+17] = ulid.Encoding[id.ULID[10]&31]
	dst[plen+18] = ulid.Encoding[(id.ULID[11]&248)>>3]
	dst[plen+19] = ulid.Encoding[((id.ULID[11]&7)<<2)|((id.ULID[12]&192)>>6)]
	dst[plen+20] = ulid.Encoding[(id.ULID[12]&62)>>1]
	dst[plen+21] = ulid.Encoding[((id.ULID[12]&1)<<4)|((id.ULID[13]&240)>>4)]
	dst[plen+22] = ulid.Encoding[((id.ULID[13]&15)<<1)|((id.ULID[14]&128)>>7)]
	dst[plen+23] = ulid.Encoding[(id.ULID[14]&124)>>2]

	return nil
}

// MarshalText implements the encoding.TextMarshaler interface by
// returning the string encoded ULID with the short ident prefix and
// without the two last bytes (since they are redundant with the prefix).
func (id ID[T]) MarshalText() ([]byte, error) {
	dst := make([]byte, id.EncodedSize())

	return dst, id.MarshalTextTo(dst)
}

// UnmarshalText implements the encoding.TextUnmarshaler interface by
// parsing the data as string encoded ULID while requiring the short ident as prefix.
func (id *ID[T]) UnmarshalText(v []byte) error {
	var kind T
	var suffix [2]byte
	binary.BigEndian.PutUint16(suffix[:], kind.KindNumber())

	before, after, found := bytes.Cut(v, []byte(kind.KindShortIdent()+"_"))
	if !found && len(before) == ulid.EncodedSize {
		if err := id.ULID.UnmarshalText(before); err != nil {
			return err //nolint:wrapcheck
		}

		if id.ULID[14] != suffix[0] || id.ULID[15] != suffix[1] {
			return ErrInvalidSuffix
		}

		return nil
	} else if !found {
		return ErrNoPrefix
	}

	return id.ULID.UnmarshalText(append(after, suffix[:]...)) //nolint:wrapcheck
}

// Kind describes the entity kind.
type Kind interface {
	KindNumber() uint16
	KindIdent() string
	KindShortIdent() string
}

// Make generates a new self-describing ULID.
func Make[T Kind]() (id ID[T]) {
	id.ULID = ulid.Make()
	id.putSuffixBytes()

	return
}

// MustFromULID parses s as a ULID but sets the trailing two bytes to make it describe T.
func MustFromULID[T Kind](s string) (id ID[T]) {
	id, err := FromULID[T](s)
	if err != nil {
		panic(err)
	}

	return id
}

// FromULID parses s as a ULID while erroring if the ulid parsing fails.
func FromULID[T Kind](s string) (id ID[T], err error) {
	id.ULID, err = ulid.Parse(s)
	if err != nil {
		return id, fmt.Errorf("failed to parse ulid: %w", err)
	}

	id.putSuffixBytes()

	return
}

// DomainSQL generates SQL for a PostgreSQL domain that constrains the ID
// by checking the length and the 2-byte suffix for the entity type.
func DomainSQL[T Kind]() string {
	var kind T

	return fmt.Sprintf(`
		CREATE DOMAIN %s_id AS bytea 
		CHECK (
			octet_length(VALUE) = 16 AND 
			get_byte(VALUE, 14) = %d AND 
			get_byte(VALUE, 15) = %d
		)`,
		kind.KindIdent(),
		kind.KindNumber()>>8,   //nolint:mnd
		kind.KindNumber()&0xFF, //nolint:mnd
	)
}
