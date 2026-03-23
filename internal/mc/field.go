package mc

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/errutil"
	"github.com/google/uuid"
)

type Field interface {
	io.ReaderFrom
	io.WriterTo
}

type FieldPtr[T any] interface {
	*T
	Field
}

type (
	// Boolean Encodes:
	//  - Either false or true.
	// Size:
	//  - 1 byte
	// Notes:
	//  - True is encoded as 0x01, false as 0x00.
	Boolean bool
	// Byte Encodes:
	//  - An integer between -128 and 127.
	// Size:
	//  - 1 byte
	// Notes:
	//  - Signed 8-bit integer, two's complement.
	Byte int8
	// UnsignedByte Encodes:
	//  - An integer between 0 and 255.
	// Size:
	//  - 1 byte
	// Notes:
	//  - Unsigned 8-bit integer.
	UnsignedByte uint8
	// Short Encodes:
	//  - An integer between -32768 and 32767.
	// Size:
	//  - 2 bytes
	// Notes:
	//  - Signed 16-bit integer, two's complement.
	Short int16
	// UnsignedShort Encodes:
	//  - An integer between 0 and 65535.
	// Size:
	//  - 2 bytes
	// Notes:
	//  - Unsigned 16-bit integer
	UnsignedShort uint16
	// Int Encodes:
	//  - An integer between -2147483648 and 2147483647.
	// Size:
	//  - 4 bytes
	// Notes:
	//  - Signed 32-bit integer, two's complement.
	Int int32
	// Long Encodes:
	//  - An integer between -9223372036854775808 and 9223372036854775807.
	// Size:
	//  - 8 bytes
	// Notes:
	//  - Signed 64-bit integer, two's complement.
	Long int64
	// Float Encodes:
	//  - A single-precision 32-bit IEEE 754 floating point number.
	// Size:
	//  - 4 bytes
	Float float32
	// Double Encodes:
	//  - A double-precision 64-bit IEEE 754 floating point number.
	// Size:
	//  - 8 bytes
	Double float64
	// String (n) Encodes:
	//  - A sequence of Unicode scalar values.
	// Size:
	//  - ≥ 1 ≤ (n×3) + 3 bytes
	// Notes:
	//  - UTF-8 string prefixed with its size in bytes as a VarInt. Maximum length of n characters, which varies by context; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#Type:String.
	String        string
	String256     string
	String16      string
	BoundedString struct {
		Value     string
		MaxLength int32
	}
	// Identifier Encodes:
	//  - A namespaced identifier; https://minecraft.wiki/w/Java_Edition_protocol/Packets#Identifier.
	// Size:
	//  - ≥ 1 and ≤ (32767×3) + 3
	// Notes:
	//  - Encoded as a String with max length of 32767.
	Identifier string
	// VarInt (n) Encodes:
	//  - An integer between -2147483648 and 2147483647.
	// Size:
	//  - ≥ 1 ≤ 5 bytes
	// Notes:
	//  - Variable-length data encoding a two's complement signed 32-bit integer; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#VarInt_and_VarLong.
	VarInt int32
	// Angle Encodes:
	//  - A rotation angle in steps of 1/256 of a full turn
	// Size:
	//  - 1 byte
	// Notes:
	//  - Whether or not this is signed does not matter, since the resulting angles are the same.
	Angle = UnsignedByte
	// UUID Encodes:
	//  - A UUID; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#Type:UUID.
	// Size:
	//  - 16 bytes
	// Notes:
	//  - Encoded as an unsigned 128-bit integer (or two unsigned 64-bit integers: the most significant 64 bits and then the least significant 64 bits).
	UUID uuid.UUID
	// LpVec3 Encodes:
	//  - https://minecraft.wiki/w/Java_Edition_protocol/Data_types#LpVec3.
	// Size:
	//  - Varies
	// Notes:
	//  - Usually used for low velocities.
	LpVec3 struct {
		X, Y, Z float64
	}
	// Position Encodes:
	//  - An integer/block position: x (-33554432 to 33554431), z (-33554432 to 33554431), y (-2048 to 2047)
	// Size:
	//  - 8 bytes
	// Notes:
	//  - Encoded as a single 64-bit integer, with the x, y, and z coordinates packed into it. The x coordinate is stored in the most significant 26 bits, the z coordinate in the next 26 bits, and the y coordinate in the least significant 12 bits.
	Position struct {
		X int32
		Y int32
		Z int32
	}
)

type Coordinate [3]Double

func NewCoordinate(pos [3]float64) Coordinate {
	return Coordinate{Double(pos[0]), Double(pos[1]), Double(pos[2])}
}

func (p Coordinate) WriteTo(w io.Writer) (n int64, err error) {
	for _, d := range p {
		nn, err := d.WriteTo(w)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func DegreesToAngle(degrees float32) Angle {
	return Angle(degrees / 360.0 * 256.0)
}

func (v VarInt) Len() int {
	val := uint32(v)

	if v < 0 {
		return 5
	}
	n := 1
	for val >= 0x80 {
		val >>= 7
		n++
	}
	return n
}

const (
	MaxQuantizedValue = 32766.0
)

func pack(value float64) int64 {
	return int64(math.Round((value*0.5 + 0.5) * MaxQuantizedValue))
}

func unpack(value uint64) float64 {
	v := float64(value & 32767)
	v = math.Min(v, MaxQuantizedValue)

	return v*2.0/MaxQuantizedValue - 1.0
}

func (b *Boolean) ReadFrom(r io.Reader) (n int64, err error) {
	var val byte
	if br, ok := r.(io.ByteReader); ok {
		val, err = br.ReadByte()
	} else {
		var buf [1]byte
		_, err = io.ReadFull(r, buf[:])
		val = buf[0]
	}
	if err != nil {
		return n, errutil.WrapIOErr(err, "error reading Boolean")
	}
	*b = val != 0
	return 1, nil
}

func (b Boolean) WriteTo(w io.Writer) (n int64, err error) {
	val := byte(0x00)
	if b {
		val = 0x01
	}
	if bw, ok := w.(io.ByteWriter); ok {
		if err := bw.WriteByte(val); err != nil {
			return 0, errutil.WrapIOErr(err, "error writing Boolean")
		}
		return 1, nil
	}
	var buf [1]byte
	buf[0] = val
	if _, err = w.Write(buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error writing Boolean")
	}
	return 1, nil
}

func (b *Byte) ReadFrom(r io.Reader) (n int64, err error) {
	if br, ok := r.(io.ByteReader); ok {
		val, err := br.ReadByte()
		if err != nil {
			return 0, errutil.WrapIOErr(err, "error reading Byte")
		}
		*b = Byte(val)
		return 1, nil
	}
	var buf [1]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error reading Byte")
	}
	*b = Byte(buf[0])
	return 1, nil
}

func (b Byte) WriteTo(w io.Writer) (n int64, err error) {
	if bw, ok := w.(io.ByteWriter); ok {
		if err := bw.WriteByte(byte(b)); err != nil {
			return 0, errutil.WrapIOErr(err, "error writing Byte")
		}
		return 1, nil
	}
	var buf [1]byte
	buf[0] = byte(b)
	if _, err = w.Write(buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error writing Byte")
	}
	return 1, nil
}

func (u *UnsignedByte) ReadFrom(r io.Reader) (n int64, err error) {
	if br, ok := r.(io.ByteReader); ok {
		val, err := br.ReadByte()
		if err != nil {
			return 0, errutil.WrapIOErr(err, "error reading UnsignedByte")
		}
		*u = UnsignedByte(val)
		return 1, nil
	}
	var buf [1]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error reading UnsignedByte")
	}
	*u = UnsignedByte(buf[0])
	return 1, nil
}

func (u UnsignedByte) WriteTo(w io.Writer) (n int64, err error) {
	if bw, ok := w.(io.ByteWriter); ok {
		if err := bw.WriteByte(byte(u)); err != nil {
			return 0, errutil.WrapIOErr(err, "error writing UnsignedByte")
		}
		return 1, nil
	}
	var buf [1]byte
	buf[0] = byte(u)
	if _, err = w.Write(buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error writing UnsignedByte")
	}
	return 1, nil
}

func (s *Short) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error reading Short")
	}
	*s = Short(binary.BigEndian.Uint16(buf[:]))
	return 2, nil
}

func (s Short) WriteTo(w io.Writer) (n int64, err error) {
	var buf [2]byte
	b := binary.BigEndian.AppendUint16(buf[:0], uint16(s))
	nn, err := w.Write(b)
	if err != nil {
		return int64(nn), errutil.WrapIOErr(err, "error writing Short")
	}
	return int64(nn), nil
}

func (u *UnsignedShort) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error reading UnsignedShort")
	}
	*u = UnsignedShort(binary.BigEndian.Uint16(buf[:]))
	return 2, nil
}

func (u UnsignedShort) WriteTo(w io.Writer) (n int64, err error) {
	var buf [2]byte
	b := binary.BigEndian.AppendUint16(buf[:0], uint16(u))
	nn, err := w.Write(b)
	if err != nil {
		return int64(nn), errutil.WrapIOErr(err, "error writing UnsignedShort")
	}
	return int64(nn), nil
}

func (i *Int) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [4]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error reading Int")
	}
	*i = Int(binary.BigEndian.Uint32(buf[:]))
	return 4, nil
}

func (i Int) WriteTo(w io.Writer) (n int64, err error) {
	var buf [4]byte
	b := binary.BigEndian.AppendUint32(buf[:0], uint32(i))
	nn, err := w.Write(b)
	if err != nil {
		return int64(nn), errutil.WrapIOErr(err, "error writing Int")
	}
	return int64(nn), nil
}

func (l *Long) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [8]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error reading Long")
	}
	*l = Long(binary.BigEndian.Uint64(buf[:]))
	return 8, nil
}

func (l Long) WriteTo(w io.Writer) (n int64, err error) {
	var buf [8]byte
	b := binary.BigEndian.AppendUint64(buf[:0], uint64(l))
	nn, err := w.Write(b)
	if err != nil {
		return int64(nn), errutil.WrapIOErr(err, "error writing Long")
	}
	return int64(nn), nil
}

func (f *Float) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [4]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error reading Float")
	}
	*f = Float(math.Float32frombits(binary.BigEndian.Uint32(buf[:])))
	return 4, nil
}

func (f Float) WriteTo(w io.Writer) (n int64, err error) {
	var buf [4]byte
	b := binary.BigEndian.AppendUint32(buf[:0], math.Float32bits(float32(f)))
	nn, err := w.Write(b)
	if err != nil {
		return int64(nn), errutil.WrapIOErr(err, "error writing Float")
	}
	return int64(nn), nil
}

func (d *Double) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [8]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, errutil.WrapIOErr(err, "error reading Double")
	}
	*d = Double(math.Float64frombits(binary.BigEndian.Uint64(buf[:])))
	return 8, nil
}

func (d Double) WriteTo(w io.Writer) (n int64, err error) {
	var buf [8]byte
	b := binary.BigEndian.AppendUint64(buf[:0], math.Float64bits(float64(d)))
	nn, err := w.Write(b)
	if err != nil {
		return int64(nn), errutil.WrapIOErr(err, "error writing Double")
	}
	return int64(nn), nil
}

func encodeString(w io.Writer, s string) (n int64, err error) {
	length := VarInt(len(s))
	n, err = length.WriteTo(w)
	if err != nil {
		return n, err
	}
	nStr, err := io.WriteString(w, s)
	if err != nil {
		return n + int64(nStr), errutil.WrapIOErr(err, "error writing String")
	}
	return n + int64(nStr), nil
}

func decodeString(r io.Reader, target *string, maxN int32) (n int64, err error) {
	var byteLength VarInt
	nn, err := byteLength.ReadFrom(r)
	if err != nil {
		return n, err
	}
	n += nn
	maxBytes := maxN * 3
	if byteLength < 0 || int32(byteLength) > maxBytes {
		return n, fmt.Errorf("string byteLength %d is out of bounds (must be between 0 and %d)", byteLength, maxBytes)
	}
	buf := make([]byte, byteLength)
	readBytes, err := io.ReadFull(r, buf)
	if err != nil {
		return n + int64(readBytes), errutil.WrapIOErr(err, "error reading String bytes")
	}
	n += int64(readBytes)
	str := string(buf)
	var codeUnits int32 = 0
	for _, char := range str {
		if char > 0xFFFF {
			codeUnits += 2
		} else {
			codeUnits += 1
		}
	}
	if codeUnits > maxN {
		return n, fmt.Errorf("string code unit length %d is out of bounds (must be between 0 and %d)", codeUnits, maxN)
	}
	*target = str
	return n, nil
}

func (s *String) ReadFrom(r io.Reader) (n int64, err error) {
	return decodeString(r, (*string)(s), 32767)
}

func (s String) WriteTo(w io.Writer) (n int64, err error) {
	return encodeString(w, string(s))
}

func (s *String256) ReadFrom(r io.Reader) (n int64, err error) {
	return decodeString(r, (*string)(s), 256)
}

func (s String256) WriteTo(w io.Writer) (n int64, err error) {
	return encodeString(w, string(s))
}

func (s *String16) ReadFrom(r io.Reader) (n int64, err error) {
	return decodeString(r, (*string)(s), 16)
}

func (s String16) WriteTo(w io.Writer) (n int64, err error) {
	return encodeString(w, string(s))
}

func (b *BoundedString) ReadFrom(r io.Reader) (n int64, err error) {
	limit := b.MaxLength
	if limit <= 0 {
		limit = 32767
	}
	return decodeString(r, &b.Value, limit)
}

func (b *BoundedString) WriteTo(w io.Writer) (n int64, err error) {
	return encodeString(w, b.Value)
}

func (v *VarInt) ReadFrom(r io.Reader) (n int64, err error) {
	var position uint
	*v = 0
	if br, ok := r.(io.ByteReader); ok {
		for i := 0; i < 5; i++ {
			b, err := br.ReadByte()
			if err != nil {
				return n, errutil.WrapIOErr(err, "error reading VarInt")
			}
			n++
			*v |= VarInt(b&0x7F) << position
			if b&0x80 == 0 {
				return n, nil
			}
			position += 7
		}
		return n, fmt.Errorf("error reading VarInt")
	}
	for i := 0; i < 5; i++ {
		var b [1]byte
		if _, err = io.ReadFull(r, b[:]); err != nil {
			return n, errutil.WrapIOErr(err, "error reading VarInt")
		}
		n++
		*v |= VarInt(b[0]&0x7F) << position
		if b[0]&0x80 == 0 {
			return n, nil
		}
		position += 7
	}
	return n, fmt.Errorf("error reading VarInt")
}

func (v VarInt) WriteTo(w io.Writer) (n int64, err error) {
	var buf [5]byte
	i := 0
	val := uint32(v)
	for {
		buf[i] = byte(val & 0x7F)
		val >>= 7
		if val != 0 {
			buf[i] |= 0x80
		}
		i++
		if val == 0 {
			break
		}
	}
	nn, err := w.Write(buf[:i])
	if err != nil {
		return int64(nn), errutil.WrapIOErr(err, "error writing VarInt")
	}
	return int64(nn), nil
}

func (u *UUID) ReadFrom(r io.Reader) (n int64, err error) {
	nBytes, err := io.ReadFull(r, (*u)[:])
	if err != nil {
		return int64(nBytes), errutil.WrapIOErr(err, "error reading UUID")
	}
	return int64(nBytes), nil
}

func (u UUID) WriteTo(w io.Writer) (n int64, err error) {
	nBytes, err := w.Write(u[:])
	if err != nil {
		return int64(nBytes), errutil.WrapIOErr(err, "error writing UUID")
	}
	return int64(nBytes), nil
}

func (i *Identifier) ReadFrom(r io.Reader) (n int64, err error) {
	var value String
	n, err = value.ReadFrom(r)
	if err != nil {
		return n, err
	}
	parts := strings.Split(string(value), ":")
	count := len(parts)
	if count < 1 || count > 2 {
		return n, fmt.Errorf("invalid Identifier: too many or not enough colons in %s", value)
	}
	if count == 2 {
		if parts[0] != "minecraft" && parts[0] != "" {
			return n, fmt.Errorf("invalid Identifier: invalid namespace %s", parts[0])
		}
	}
	*i = Identifier(parts[count-1])
	for _, c := range *i {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || c == '_' || c == '-' || c == '.' || c == '/') {
			return n, fmt.Errorf("invalid Identifier: forbidden character '%c' in path %s", c, *i)
		}
	}
	return n, nil
}

func (i Identifier) WriteTo(w io.Writer) (n int64, err error) {
	// NOTE: we only support the default namespace
	return String("minecraft:" + i).WriteTo(w)
}

func (p *Position) ReadFrom(r io.Reader) (n int64, err error) {
	var val Long
	n, err = val.ReadFrom(r)
	if err != nil {
		return
	}
	p.X = int32(val >> 38)
	p.Y = int32(val << 52 >> 52)
	p.Z = int32(val << 26 >> 38)
	return
}

func (p Position) WriteTo(w io.Writer) (n int64, err error) {
	val := ((int64(p.X) & 0x3FFFFFF) << 38) | (int64(p.Y) & 0xFFF) | ((int64(p.Z) & 0x3FFFFFF) << 12)
	return Long(val).WriteTo(w)
}

func (l *LpVec3) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return n, err
	}
	n += 1
	byte1 := uint64(buf[0])
	if byte1 == 0 {
		l.X, l.Y, l.Z = 0, 0, 0
		return n, nil
	}
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return n, err
	}
	n += 1
	byte2 := uint64(buf[0])
	var fourBytes [4]byte
	if _, err := io.ReadFull(r, fourBytes[:]); err != nil {
		return n, err
	}
	n += 4
	bytes3To4 := binary.BigEndian.Uint32(fourBytes[:])
	packed := (uint64(bytes3To4) << 16) | (byte2 << 8) | byte1
	scaleFactor := byte1 & 3
	if (byte1 & 4) == 4 {
		rest := VarInt(0)
		nn, err := rest.ReadFrom(r)
		n += nn
		if err != nil {
			return n + nn, err
		}
		scaleFactor |= uint64(rest) << 2
	}
	scaleFactorD := float64(scaleFactor)
	l.X = unpack(packed>>3) * scaleFactorD
	l.Y = unpack(packed>>18) * scaleFactorD
	l.Z = unpack(packed>>33) * scaleFactorD
	return n, nil
}

func (l LpVec3) WriteTo(w io.Writer) (n int64, err error) {
	maxCoordinate := math.Max(math.Abs(l.X), math.Max(math.Abs(l.Y), math.Abs(l.Z)))
	if maxCoordinate < 3.051944088384301e-5 {
		if _, err := w.Write([]byte{0}); err != nil {
			return 0, err
		}
		return 1, nil
	}
	maxCoordinateI := int64(maxCoordinate)
	var scaleFactor int64
	if maxCoordinate > float64(maxCoordinateI) {
		scaleFactor = maxCoordinateI + 1
	} else {
		scaleFactor = maxCoordinateI
	}
	needContinuation := (scaleFactor & 3) != scaleFactor
	var packedScale int64
	if needContinuation {
		packedScale = (scaleFactor & 3) | 4
	} else {
		packedScale = scaleFactor
	}
	scaleFactorD := float64(scaleFactor)
	packedX := pack(l.X/scaleFactorD) << 3
	packedY := pack(l.Y/scaleFactorD) << 18
	packedZ := pack(l.Z/scaleFactorD) << 33
	packed := packedZ | packedY | packedX | packedScale
	var writeBuf [6]byte
	writeBuf[0] = byte(packed)
	writeBuf[1] = byte(packed >> 8)
	valInt := uint32(packed >> 16)
	binary.BigEndian.PutUint32(writeBuf[2:], valInt)
	if _, err := w.Write(writeBuf[:]); err != nil {
		return 0, err
	}
	n += 6
	if needContinuation {
		buf := VarInt(scaleFactor >> 2)
		nn, err := buf.WriteTo(w)
		n += nn
		return n, err
	}
	return n, nil
}
