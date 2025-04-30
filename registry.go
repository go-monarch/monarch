package monarch

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	tUUID         = reflect.TypeOf(uuid.UUID{})
	uuidSubtype   = byte(0x04)
	mongoRegistry = bson.NewRegistry()
)

func init() {
	mongoRegistry.RegisterTypeEncoder(tUUID, bson.ValueEncoderFunc(uuidEncodeValue))
	mongoRegistry.RegisterTypeDecoder(tUUID, bson.ValueDecoderFunc(uuidDecodeValue))

}

func uuidEncodeValue(ec bson.EncodeContext, vw bson.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tUUID {
		return bson.ValueEncoderError{Name: "uuidEncodeValue", Types: []reflect.Type{tUUID}, Received: val}
	}
	b := val.Interface().(uuid.UUID)
	return vw.WriteBinaryWithSubtype(b[:], uuidSubtype)
}

func uuidDecodeValue(dc bson.DecodeContext, vr bson.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tUUID {
		return bson.ValueDecoderError{Name: "uuidDecodeValue", Types: []reflect.Type{tUUID}, Received: val}
	}

	var data []byte
	var subtype byte
	var err error
	switch vrType := vr.Type(); vrType {
	case bson.TypeBinary:
		data, subtype, err = vr.ReadBinary()
		if subtype != uuidSubtype {
			return fmt.Errorf("unsupported binary subtype %v for UUID", subtype)
		}
	case bson.TypeNull:
		err = vr.ReadNull()
	case bson.TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return fmt.Errorf("cannot decode %v into a UUID", vrType)
	}

	if err != nil {
		return err
	}
	uuid2, err := uuid.FromBytes(data)
	if err != nil {
		return err
	}
	val.Set(reflect.ValueOf(uuid2))
	return nil
}
