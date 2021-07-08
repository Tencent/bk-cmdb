package monstachemap

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const timeJsonFormat = "2006-01-02T15:04:05.000Z07:00"

type Time struct {
	time.Time
}

type Binary struct {
	primitive.Binary
}

type Decimal128 struct {
	primitive.Decimal128
}

func (t Time) MarshalJSON() ([]byte, error) {
	if y := t.Year(); y < 0 || y >= 10000 {
		return nil, errors.New("Time.MarshalJSON: year outside of range [0,9999]")
	}
	b := make([]byte, 0, len(timeJsonFormat)+2)
	b = append(b, '"')
	b = t.AppendFormat(b, timeJsonFormat)
	b = append(b, '"')
	return b, nil
}

func (bi Binary) MarshalJSON() ([]byte, error) {
	encoded := EncodeBinData(bi)
	b := make([]byte, 0, len(encoded)+2)
	b = append(b, '"')
	b = append(b, []byte(encoded)...)
	b = append(b, '"')
	return b, nil
}

func (dec Decimal128) MarshalJSON() ([]byte, error) {
	encoded := dec.String()
	if encoded == "NaN" ||
		strings.HasPrefix(encoded, "-Inf") ||
		strings.HasPrefix(encoded, "Inf") ||
		strings.HasPrefix(encoded, "+Inf") {
		return []byte("null"), nil
	}
	return []byte(encoded), nil
}

func EncodeBinData(bi Binary) string {
	var enc string
	if bi.Subtype == 0x03 || bi.Subtype == 0x04 {
		// UUID
		hex := hex.EncodeToString(bi.Data)
		if len(hex) == 32 {
			enc = strings.Join(
				[]string{
					hex[:8],
					hex[8:12],
					hex[12:16],
					hex[16:20],
					hex[20:32],
				},
				"-",
			)
		} else {
			enc = hex
		}
	} else {
		// other binary types
		enc = base64.StdEncoding.EncodeToString(bi.Data)
	}
	return enc
}

func ConvertSliceForJSON(a []interface{}) []interface{} {
	var avs = make([]interface{}, len(a))
	for i, av := range a {
		var avc interface{}
		switch achild := av.(type) {
		case map[string]interface{}:
			avc = ConvertMapForJSON(achild)
		case []interface{}:
			avc = ConvertSliceForJSON(achild)
		case primitive.Binary:
			avc = Binary{achild}
		case primitive.Decimal128:
			avc = Decimal128{achild}
		case time.Time:
			avc = Time{achild}
		default:
			avc = av
		}
		avs[i] = avc
	}
	return avs
}

func ConvertMapForJSON(m map[string]interface{}) map[string]interface{} {
	o := map[string]interface{}{}
	for k, v := range m {
		switch child := v.(type) {
		case map[string]interface{}:
			o[k] = ConvertMapForJSON(child)
		case []interface{}:
			o[k] = ConvertSliceForJSON(child)
		case primitive.Binary:
			o[k] = Binary{child}
		case primitive.Decimal128:
			o[k] = Decimal128{child}
		case time.Time:
			o[k] = Time{child}
		default:
			o[k] = v
		}
	}
	return o
}
