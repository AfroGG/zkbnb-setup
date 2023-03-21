package phase2

import (
	"encoding/binary"
	"io"

	"github.com/consensys/gnark-crypto/ecc/bn254"
)

type Header struct {
	Internal      uint32
	Public        uint32
	Constraints   uint32
	Domain        uint32
	Contributions uint16
	G1            struct {
		Alpha bn254.G1Affine
		Beta  bn254.G1Affine
	}
	G2 struct {
		Beta bn254.G2Affine
	}
}

func (p *Header) readFrom(reader io.Reader) error {
	// Internal
	buff := make([]byte, 4)
	if _, err := reader.Read(buff); err != nil {
		return err
	}
	p.Internal = binary.BigEndian.Uint32(buff)

	// Public
	if _, err := reader.Read(buff); err != nil {
		return err
	}
	p.Public = binary.BigEndian.Uint32(buff)

	// Constraints
	if _, err := reader.Read(buff); err != nil {
		return err
	}
	p.Constraints = binary.BigEndian.Uint32(buff)

	// Domain
	if _, err := reader.Read(buff); err != nil {
		return err
	}
	p.Domain = binary.BigEndian.Uint32(buff)

	// Contributions
	buff = buff[:2]
	if _, err := reader.Read(buff); err != nil {
		return err
	}
	p.Contributions = binary.BigEndian.Uint16(buff)

	// G1.Alpha
	dec := bn254.NewDecoder(reader)
	if err := dec.Decode(&p.G1.Alpha); err != nil {
		return err
	}

	// G1.Beta
	if err := dec.Decode(&p.G1.Beta); err != nil {
		return err
	}

	// G2.Beta
	if err := dec.Decode(&p.G2.Beta); err != nil {
		return err
	}

	return nil
}

func (p *Header) writeTo(writer io.Writer) error {
	// Internal
	buff := make([]byte, 4)
	binary.BigEndian.PutUint32(buff, p.Internal)
	if _, err := writer.Write(buff); err != nil {
		return err
	}

	// Public
	binary.BigEndian.PutUint32(buff, p.Public)
	if _, err := writer.Write(buff); err != nil {
		return err
	}

	// Constraints
	binary.BigEndian.PutUint32(buff, p.Constraints)
	if _, err := writer.Write(buff); err != nil {
		return err
	}

	// Domain
	binary.BigEndian.PutUint32(buff, p.Domain)
	if _, err := writer.Write(buff); err != nil {
		return err
	}

	// Contributions
	buff = buff[:2]
	binary.BigEndian.PutUint16(buff, p.Contributions)
	if _, err := writer.Write(buff); err != nil {
		return err
	}

	// G1.Alpha
	enc := bn254.NewEncoder(writer)
	if err := enc.Encode(&p.G1.Alpha); err != nil {
		return err
	}

	// G1.Beta
	if err := enc.Encode(&p.G1.Beta); err != nil {
		return err
	}

	// G2.Beta
	if err := enc.Encode(&p.G2.Beta); err != nil {
		return err
	}

	return nil
}
