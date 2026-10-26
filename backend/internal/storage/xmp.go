package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"strings"
)

const syntheticPerformerKeyword = "contains-synthetic-performer"

var (
	pngSignature  = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	jpegSignature = []byte{0xFF, 0xD8}
)

// EnsureSyntheticPerformerXMP adds the marketplace-required keyword to the
// XMP dc:subject array. Unsupported formats are returned unchanged.
func EnsureSyntheticPerformerXMP(data []byte, mimeType string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	mimeType = strings.ToLower(strings.TrimSpace(strings.SplitN(mimeType, ";", 2)[0]))
	switch {
	case mimeType == "image/png" || bytes.HasPrefix(data, pngSignature):
		return addPNGXMP(data)
	case mimeType == "image/jpeg" || mimeType == "image/jpg" || bytes.HasPrefix(data, jpegSignature):
		return addJPEGXMP(data)
	case mimeType == "image/webp" || isWebP(data):
		return addWebPXMP(data)
	default:
		return data, nil
	}
}

func syntheticPerformerXMPPacket() []byte {
	return []byte("<?xpacket begin=\"\xEF\xBB\xBF\" id=\"W5M0MpCehiHzreSzNTczkc9d\"?>" +
		"<x:xmpmeta xmlns:x=\"adobe:ns:meta/\">" +
		"<rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">" +
		"<rdf:Description rdf:about=\"\" xmlns:dc=\"http://purl.org/dc/elements/1.1/\">" +
		"<dc:subject><rdf:Bag><rdf:li>" + syntheticPerformerKeyword +
		"</rdf:li></rdf:Bag></dc:subject>" +
		"</rdf:Description></rdf:RDF></x:xmpmeta>" +
		"<?xpacket end=\"w\"?>")
}

func addKeywordToXMP(packet []byte) ([]byte, bool) {
	if bytes.Contains(packet, []byte(syntheticPerformerKeyword)) {
		return packet, true
	}
	if subject := bytes.Index(packet, []byte("<dc:subject")); subject >= 0 {
		if bagEnd := bytes.Index(packet[subject:], []byte("</rdf:Bag>")); bagEnd >= 0 {
			insertAt := subject + bagEnd
			return insertBytes(packet, insertAt, []byte("<rdf:li>"+syntheticPerformerKeyword+"</rdf:li>")), true
		}
	}
	if descriptionEnd := bytes.Index(packet, []byte("</rdf:Description>")); descriptionEnd >= 0 {
		subject := []byte("<dc:subject xmlns:dc=\"http://purl.org/dc/elements/1.1/\"><rdf:Bag><rdf:li>" + syntheticPerformerKeyword + "</rdf:li></rdf:Bag></dc:subject>")
		return insertBytes(packet, descriptionEnd, subject), true
	}
	if rdfEnd := bytes.Index(packet, []byte("</rdf:RDF>")); rdfEnd >= 0 {
		description := []byte("<rdf:Description rdf:about=\"\" xmlns:dc=\"http://purl.org/dc/elements/1.1/\"><dc:subject><rdf:Bag><rdf:li>" + syntheticPerformerKeyword + "</rdf:li></rdf:Bag></dc:subject></rdf:Description>")
		return insertBytes(packet, rdfEnd, description), true
	}
	return packet, false
}

func insertBytes(data []byte, offset int, value []byte) []byte {
	result := make([]byte, 0, len(data)+len(value))
	result = append(result, data[:offset]...)
	result = append(result, value...)
	result = append(result, data[offset:]...)
	return result
}

func addPNGXMP(data []byte) ([]byte, error) {
	if !bytes.HasPrefix(data, pngSignature) {
		return nil, fmt.Errorf("invalid PNG signature")
	}
	insertAt := len(data)
	for offset := len(pngSignature); offset+12 <= len(data); {
		length := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		end := offset + 12 + length
		if length < 0 || end > len(data) {
			return nil, fmt.Errorf("invalid PNG chunk length")
		}
		typ := string(data[offset+4 : offset+8])
		if typ == "iTXt" {
			payload := data[offset+8 : offset+8+length]
			const prefix = "XML:com.adobe.xmp\x00\x00\x00\x00\x00"
			if bytes.HasPrefix(payload, []byte(prefix)) {
				if packet, ok := addKeywordToXMP(payload[len(prefix):]); ok {
					updated := append([]byte(prefix), packet...)
					return replaceBytes(data, offset, end, pngChunk("iTXt", updated)), nil
				}
			}
		}
		if typ == "IDAT" && insertAt == len(data) {
			insertAt = offset
		}
		if typ == "IEND" {
			if insertAt == len(data) {
				insertAt = offset
			}
			break
		}
		offset = end
	}
	payload := append([]byte("XML:com.adobe.xmp"), []byte{0, 0, 0, 0, 0}...)
	payload = append(payload, syntheticPerformerXMPPacket()...)
	chunk := pngChunk("iTXt", payload)

	result := make([]byte, 0, len(data)+len(chunk))
	result = append(result, data[:insertAt]...)
	result = append(result, chunk...)
	result = append(result, data[insertAt:]...)
	return result, nil
}

func replaceBytes(data []byte, start, end int, value []byte) []byte {
	result := make([]byte, 0, len(data)-(end-start)+len(value))
	result = append(result, data[:start]...)
	result = append(result, value...)
	result = append(result, data[end:]...)
	return result
}

func pngChunk(typ string, payload []byte) []byte {
	result := make([]byte, 12+len(payload))
	binary.BigEndian.PutUint32(result[:4], uint32(len(payload)))
	copy(result[4:8], typ)
	copy(result[8:8+len(payload)], payload)
	binary.BigEndian.PutUint32(result[8+len(payload):], crc32.ChecksumIEEE(result[4:8+len(payload)]))
	return result
}

func addJPEGXMP(data []byte) ([]byte, error) {
	if !bytes.HasPrefix(data, jpegSignature) {
		return nil, fmt.Errorf("invalid JPEG signature")
	}
	const identifier = "http://ns.adobe.com/xap/1.0/"
	prefix := append([]byte(identifier), 0)
	insertAt := 2
	for offset := 2; offset+4 <= len(data) && data[offset] == 0xFF; {
		marker := data[offset+1]
		if marker == 0xD9 || marker == 0xDA {
			break
		}
		length := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		end := offset + 2 + length
		if length < 2 || end > len(data) {
			return nil, fmt.Errorf("invalid JPEG segment length")
		}
		payload := data[offset+4 : end]
		if marker == 0xE1 && bytes.HasPrefix(payload, prefix) {
			if packet, ok := addKeywordToXMP(payload[len(prefix):]); ok {
				updated := append(append([]byte{}, prefix...), packet...)
				if len(updated)+2 > 0xFFFF {
					return nil, fmt.Errorf("XMP packet is too large")
				}
				segment := make([]byte, 4+len(updated))
				segment[0], segment[1] = 0xFF, 0xE1
				binary.BigEndian.PutUint16(segment[2:4], uint16(len(updated)+2))
				copy(segment[4:], updated)
				return replaceBytes(data, offset, end, segment), nil
			}
		}
		if marker == 0xE0 || marker == 0xE1 {
			insertAt = end
		}
		offset = end
	}
	payload := append([]byte{}, prefix...)
	payload = append(payload, syntheticPerformerXMPPacket()...)
	if len(payload)+2 > 0xFFFF {
		return nil, fmt.Errorf("XMP packet is too large")
	}
	segment := make([]byte, 4+len(payload))
	segment[0], segment[1] = 0xFF, 0xE1
	binary.BigEndian.PutUint16(segment[2:4], uint16(len(payload)+2))
	copy(segment[4:], payload)
	result := make([]byte, 0, len(data)+len(segment))
	result = append(result, data[:insertAt]...)
	result = append(result, segment...)
	result = append(result, data[insertAt:]...)
	return result, nil
}

func isWebP(data []byte) bool {
	return len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP"))
}

func addWebPXMP(data []byte) ([]byte, error) {
	if !isWebP(data) {
		return nil, fmt.Errorf("invalid WebP signature")
	}
	if len(data) < 30 || !bytes.Equal(data[12:16], []byte("VP8X")) {
		width, height, err := simpleWebPDimensions(data)
		if err != nil {
			return nil, err
		}
		vp8x := make([]byte, 18)
		copy(vp8x[:4], "VP8X")
		binary.LittleEndian.PutUint32(vp8x[4:8], 10)
		vp8x[8] = 0x04
		putUint24LE(vp8x[12:15], width-1)
		putUint24LE(vp8x[15:18], height-1)
		data = insertBytes(data, 12, vp8x)
		binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	}
	for offset := 12; offset+8 <= len(data); {
		length := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		end := offset + 8 + length + length%2
		if length < 0 || end > len(data) {
			return nil, fmt.Errorf("invalid WebP chunk length")
		}
		if bytes.Equal(data[offset:offset+4], []byte("XMP ")) {
			if packet, ok := addKeywordToXMP(data[offset+8 : offset+8+length]); ok {
				chunk := webPChunk(packet)
				result := replaceBytes(data, offset, end, chunk)
				binary.LittleEndian.PutUint32(result[4:8], uint32(len(result)-8))
				result[20] |= 0x04
				return result, nil
			}
		}
		offset = end
	}
	packet := syntheticPerformerXMPPacket()
	chunk := webPChunk(packet)

	result := append([]byte(nil), data...)
	result = append(result, chunk...)
	binary.LittleEndian.PutUint32(result[4:8], uint32(len(result)-8))
	result[20] |= 0x04
	return result, nil
}

func simpleWebPDimensions(data []byte) (uint32, uint32, error) {
	if len(data) < 20 {
		return 0, 0, fmt.Errorf("invalid WebP image data")
	}
	payloadLength := int(binary.LittleEndian.Uint32(data[16:20]))
	if payloadLength < 0 || 20+payloadLength > len(data) {
		return 0, 0, fmt.Errorf("invalid WebP image chunk length")
	}
	switch string(data[12:16]) {
	case "VP8 ":
		if payloadLength < 10 || !bytes.Equal(data[23:26], []byte{0x9D, 0x01, 0x2A}) {
			return 0, 0, fmt.Errorf("invalid VP8 frame header")
		}
		width := uint32(binary.LittleEndian.Uint16(data[26:28]) & 0x3FFF)
		height := uint32(binary.LittleEndian.Uint16(data[28:30]) & 0x3FFF)
		if width == 0 || height == 0 {
			return 0, 0, fmt.Errorf("invalid VP8 dimensions")
		}
		return width, height, nil
	case "VP8L":
		if payloadLength < 5 || data[20] != 0x2F {
			return 0, 0, fmt.Errorf("invalid VP8L frame header")
		}
		bits := binary.LittleEndian.Uint32(data[21:25])
		return (bits & 0x3FFF) + 1, ((bits >> 14) & 0x3FFF) + 1, nil
	default:
		return 0, 0, fmt.Errorf("unsupported WebP image chunk")
	}
}

func putUint24LE(dst []byte, value uint32) {
	dst[0] = byte(value)
	dst[1] = byte(value >> 8)
	dst[2] = byte(value >> 16)
}

func webPChunk(packet []byte) []byte {
	chunk := make([]byte, 8+len(packet))
	copy(chunk[:4], []byte("XMP "))
	binary.LittleEndian.PutUint32(chunk[4:8], uint32(len(packet)))
	copy(chunk[8:], packet)
	if len(packet)%2 != 0 {
		chunk = append(chunk, 0)
	}

	return chunk
}
