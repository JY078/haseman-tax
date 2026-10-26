package storage

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestEnsureSyntheticPerformerXMPPNG(t *testing.T) {
	data := append(append([]byte{}, pngSignature...), pngChunk("IEND", nil)...)
	annotated, err := EnsureSyntheticPerformerXMP(data, "image/png")
	if err != nil {
		t.Fatalf("annotate PNG: %v", err)
	}
	if !bytes.Contains(annotated, []byte("XML:com.adobe.xmp")) || !bytes.Contains(annotated, []byte("dc:subject")) || !bytes.Contains(annotated, []byte(syntheticPerformerKeyword)) {
		t.Fatal("PNG does not contain the required XMP dc:subject keyword")
	}
	again, err := EnsureSyntheticPerformerXMP(annotated, "image/png")
	if err != nil || !bytes.Equal(annotated, again) {
		t.Fatal("XMP annotation is not idempotent")
	}
}

func TestEnsureSyntheticPerformerXMPJPEG(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	annotated, err := EnsureSyntheticPerformerXMP(data, "image/jpeg")
	if err != nil {
		t.Fatalf("annotate JPEG: %v", err)
	}
	if !bytes.Contains(annotated, []byte("http://ns.adobe.com/xap/1.0/\x00")) || !bytes.Contains(annotated, []byte(syntheticPerformerKeyword)) {
		t.Fatal("JPEG does not contain the required XMP packet")
	}
}

func TestEnsureSyntheticPerformerXMPPreservesExistingSubject(t *testing.T) {
	existing := []byte("<x:xmpmeta><rdf:RDF><rdf:Description><dc:subject><rdf:Bag><rdf:li>existing-keyword</rdf:li></rdf:Bag></dc:subject></rdf:Description></rdf:RDF></x:xmpmeta>")
	payload := append([]byte("http://ns.adobe.com/xap/1.0/\x00"), existing...)
	segment := make([]byte, 4+len(payload))
	segment[0], segment[1] = 0xFF, 0xE1
	binary.BigEndian.PutUint16(segment[2:4], uint16(len(payload)+2))
	copy(segment[4:], payload)
	data := append([]byte{0xFF, 0xD8}, segment...)
	data = append(data, 0xFF, 0xD9)

	annotated, err := EnsureSyntheticPerformerXMP(data, "image/jpeg")
	if err != nil {
		t.Fatalf("annotate JPEG with existing XMP: %v", err)
	}
	if bytes.Count(annotated, []byte("http://ns.adobe.com/xap/1.0/\x00")) != 1 || !bytes.Contains(annotated, []byte("existing-keyword")) || !bytes.Contains(annotated, []byte(syntheticPerformerKeyword)) {
		t.Fatal("existing XMP dc:subject was not preserved and updated")
	}
}

func TestEnsureSyntheticPerformerXMPUpdatesSelfClosingDescription(t *testing.T) {
	existing := []byte("<x:xmpmeta><rdf:RDF><rdf:Description rdf:about=\"\"/></rdf:RDF></x:xmpmeta>")
	payload := append([]byte("http://ns.adobe.com/xap/1.0/\x00"), existing...)
	segment := make([]byte, 4+len(payload))
	segment[0], segment[1] = 0xFF, 0xE1
	binary.BigEndian.PutUint16(segment[2:4], uint16(len(payload)+2))
	copy(segment[4:], payload)
	data := append([]byte{0xFF, 0xD8}, segment...)
	data = append(data, 0xFF, 0xD9)

	annotated, err := EnsureSyntheticPerformerXMP(data, "image/jpeg")
	if err != nil {
		t.Fatalf("annotate self-closing XMP: %v", err)
	}
	if bytes.Count(annotated, []byte("http://ns.adobe.com/xap/1.0/\x00")) != 1 || !bytes.Contains(annotated, []byte("dc:subject")) || !bytes.Contains(annotated, []byte(syntheticPerformerKeyword)) {
		t.Fatal("self-closing XMP description was not updated in the existing packet")
	}
}

func TestEnsureSyntheticPerformerXMPWebP(t *testing.T) {
	data := make([]byte, 20)
	copy(data[:4], "RIFF")
	copy(data[8:12], "WEBP")
	copy(data[12:16], "VP8X")
	binary.LittleEndian.PutUint32(data[16:20], 10)
	data = append(data, make([]byte, 10)...)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	annotated, err := EnsureSyntheticPerformerXMP(data, "image/webp")
	if err != nil {
		t.Fatalf("annotate WebP: %v", err)
	}
	if !bytes.Contains(annotated, []byte("XMP ")) || !bytes.Contains(annotated, []byte(syntheticPerformerKeyword)) || annotated[20]&0x04 == 0 {
		t.Fatal("WebP does not contain the required XMP metadata flag and packet")
	}
	if got := binary.LittleEndian.Uint32(annotated[4:8]); got != uint32(len(annotated)-8) {
		t.Fatalf("invalid WebP RIFF size: got %d want %d", got, len(annotated)-8)
	}
}

func TestEnsureSyntheticPerformerXMPSimpleLosslessWebP(t *testing.T) {
	data := make([]byte, 26)
	copy(data[:4], "RIFF")
	copy(data[8:12], "WEBP")
	copy(data[12:16], "VP8L")
	binary.LittleEndian.PutUint32(data[16:20], 5)
	data[20] = 0x2F
	data[21] = 9
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))

	annotated, err := EnsureSyntheticPerformerXMP(data, "image/webp")
	if err != nil {
		t.Fatalf("annotate simple WebP: %v", err)
	}
	if !bytes.Equal(annotated[12:16], []byte("VP8X")) || !bytes.Contains(annotated, []byte("VP8L")) || !bytes.Contains(annotated, []byte(syntheticPerformerKeyword)) {
		t.Fatal("simple WebP was not upgraded to VP8X with XMP")
	}
}

func TestEnsureSyntheticPerformerXMPUnsupportedFormat(t *testing.T) {
	data := []byte("not-an-image")
	annotated, err := EnsureSyntheticPerformerXMP(data, "image/gif")
	if err != nil || !bytes.Equal(data, annotated) {
		t.Fatal("unsupported image format should be returned unchanged")
	}
}
