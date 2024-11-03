package sdulid_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/advdv/sdulid"
	"github.com/oklog/ulid/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSDULID(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunSpecs(t, "sdulid")
}

type testID struct{}

func (testID) KindNumber() uint16     { return math.MaxUint16 }
func (testID) KindIdent() string      { return "test" }
func (testID) KindShortIdent() string { return "tst" }

var _ = Describe("model id", func() {
	var id1 sdulid.ID[testID]

	BeforeEach(func() {
		id1 = sdulid.MustFromULID[testID]("01JBRQS1J5A085FYY2M7ZXWG00")
	})

	It("should have parsed from ulid while enforcing the two trailing bytes", func() {
		Expect(id1.Bytes()).To(Equal([]byte{1, 146, 241, 124, 134, 69, 80, 16, 87, 251, 194, 161, 255, 222, 255, 255}))
	})

	It("should make new ids with the byte suffix", func() {
		id1 := sdulid.Make[testID]()
		Expect(id1.Bytes()[14:]).To(Equal([]byte{255, 255}))
	})

	Describe("text encoding", func() {
		It("should error on wrong buffer size when marshaling text", func() {
			var dst []byte
			Expect(id1.MarshalTextTo(dst)).To(MatchError(sdulid.ErrBufferSize))
		})

		It("should marshal text to", func() {
			dst := make([]byte, id1.EncodedSize())
			Expect(id1.MarshalTextTo(dst)).To(Succeed())
			Expect(string(dst)).To(Equal(`tst_01JBRQS1J5A085FYY2M7ZXXZ`))
		})

		It("should marshal text", func() {
			dst, err := id1.MarshalText()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(dst)).To(Equal(`tst_01JBRQS1J5A085FYY2M7ZXXZ`))
		})

		It("should prefix with short ident for stringer", func() {
			Expect(id1.String()).To(Equal("tst_01JBRQS1J5A085FYY2M7ZXXZ"))
		})
	})

	Describe("text decoding", func() {
		It("should decode encoding format", func() {
			s1, err := id1.MarshalText()
			Expect(err).ToNot(HaveOccurred())

			var id2 sdulid.ID[testID]
			Expect(id2.UnmarshalText(s1)).To(Succeed())

			Expect(id2.String()).To(Equal(`tst_01JBRQS1J5A085FYY2M7ZXXZ`))
		})

		It("should allow long format decoding (no prefix)", func() {
			var id2 sdulid.ID[testID]
			Expect(id2.UnmarshalText([]byte("01JBRQS1J5A085FYY2M7ZXXZZZ"))).To(Succeed())
		})

		It("should not allow long format with wrong suffix", func() {
			var id2 sdulid.ID[testID]
			Expect(id2.UnmarshalText([]byte("01JBRQS1J5A085FYY2M7ZXXZZE"))).To(MatchError(sdulid.ErrInvalidSuffix))
		})

		It("should not decode without prefix and short format", func() {
			var id2 sdulid.ID[testID]
			Expect(id2.UnmarshalText([]byte("01JBRQS1J5A085FYY2M7ZXXZ"))).To(MatchError(sdulid.ErrNoPrefix))
		})
	})

	It("should", func() {
	})

	It("should fail from invalid ulid", func() {
		Expect(func() {
			sdulid.MustFromULID[testID]("0")
		}).To(And(
			PanicWith(MatchError(ContainSubstring(`failed to parse ulid`))),
			PanicWith(MatchError(ulid.ErrDataSize))))
	})

	It("should generate domain sql", func() {
		Expect(sdulid.CreateDomainSQL[testID]()).To(Equal("\n\t\tCREATE DOMAIN test_id AS bytea \n\t\tCHECK (\n\t\t\toctet_length(VALUE) = 16 AND \n\t\t\tget_byte(VALUE, 14) = 255 AND \n\t\t\tget_byte(VALUE, 15) = 255\n\t\t)"))
	})

	It("should generate generator sql", func() {
		Expect(sdulid.CreateGeneratorSQL[testID]()).To(ContainSubstring(fmt.Sprintf(`(%d >> 8) & 255)`, math.MaxUint16)))
	})
})
