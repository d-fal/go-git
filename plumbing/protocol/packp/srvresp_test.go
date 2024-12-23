package packp

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"

	. "gopkg.in/check.v1"
)

type ServerResponseSuite struct{}

var _ = Suite(&ServerResponseSuite{})

func (s *ServerResponseSuite) TestDecodeNAK(c *C) {
	raw := "0008NAK\n"

	sr := &ServerResponse{}
	err := sr.Decode((bytes.NewBufferString(raw)), false)
	c.Assert(err, IsNil)

	c.Assert(sr.ACKs, HasLen, 0)
}

func (s *ServerResponseSuite) TestDecodeNewLine(c *C) {
	raw := "\n"

	sr := &ServerResponse{}
	err := sr.Decode(bytes.NewBufferString(raw), false)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "invalid pkt-len found.*")
}

func (s *ServerResponseSuite) TestDecodeEmpty(c *C) {
	raw := ""

	sr := &ServerResponse{}
	err := sr.Decode(bytes.NewBufferString(raw), false)
	c.Assert(err, IsNil)
}

func (s *ServerResponseSuite) TestDecodePartial(c *C) {
	raw := "000600\n"

	sr := &ServerResponse{}
	err := sr.Decode(bytes.NewBufferString(raw), false)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, fmt.Sprintf("unexpected content %q", "00"))
}

func (s *ServerResponseSuite) TestDecodeACK(c *C) {
	raw := "0031ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e5\n"

	sr := &ServerResponse{}
	err := sr.Decode(bytes.NewBufferString(raw), false)
	c.Assert(err, IsNil)

	c.Assert(sr.ACKs, HasLen, 1)
	c.Assert(sr.ACKs[0], Equals, plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
}

func (s *ServerResponseSuite) TestDecodeMultipleACK(c *C) {
	raw := "" +
		"0031ACK 1111111111111111111111111111111111111111\n" +
		"0031ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e5\n" +
		"00080PACK\n"

	sr := &ServerResponse{}
	err := sr.Decode(bytes.NewBufferString(raw), false)
	c.Assert(err, IsNil)

	c.Assert(sr.ACKs, HasLen, 2)
	c.Assert(sr.ACKs[0], Equals, plumbing.NewHash("1111111111111111111111111111111111111111"))
	c.Assert(sr.ACKs[1], Equals, plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
}

func (s *ServerResponseSuite) TestDecodeMultipleACKWithSideband(c *C) {
	raw := "" +
		"0031ACK 1111111111111111111111111111111111111111\n" +
		"0031ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e5\n" +
		"00080aaaa\n"

	sr := &ServerResponse{}
	err := sr.Decode(bytes.NewBufferString(raw), false)
	c.Assert(err, IsNil)

	c.Assert(sr.ACKs, HasLen, 2)
	c.Assert(sr.ACKs[0], Equals, plumbing.NewHash("1111111111111111111111111111111111111111"))
	c.Assert(sr.ACKs[1], Equals, plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
}

func (s *ServerResponseSuite) TestDecodeMalformed(c *C) {
	raw := "0029ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e\n"

	sr := &ServerResponse{}
	err := sr.Decode(bytes.NewBufferString(raw), false)
	c.Assert(err, NotNil)
}

func (s *ServerResponseSuite) TestDecodeMultiACK(c *C) {
	raw := "" +
		"0031ACK 1111111111111111111111111111111111111111\n" +
		"0031ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e5\n" +
		"00080PACK\n"

	sr := &ServerResponse{}
	err := sr.Decode(strings.NewReader(raw), true)
	c.Assert(err, IsNil)

	c.Assert(sr.ACKs, HasLen, 2)
	c.Assert(sr.ACKs[0], Equals, plumbing.NewHash("1111111111111111111111111111111111111111"))
	c.Assert(sr.ACKs[1], Equals, plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
}

func (s *ServerResponseSuite) TestEncodeEmpty(c *C) {
	haves := make(chan UploadPackCommand)
	go func() {
		haves <- UploadPackCommand{
			Acks: []UploadPackRequestAck{},
			Done: true,
		}
		close(haves)
	}()
	sr := &ServerResponse{req: &UploadPackRequest{UploadPackCommands: haves, UploadRequest: UploadRequest{Capabilities: capability.NewList()}}}
	b := bytes.NewBuffer(nil)
	err := sr.Encode(b)
	c.Assert(err, IsNil)

	c.Assert(b.String(), Equals, "0008NAK\n")
}

func (s *ServerResponseSuite) TestEncodeSingleAck(c *C) {
	haves := make(chan UploadPackCommand)
	go func() {
		haves <- UploadPackCommand{
			Acks: []UploadPackRequestAck{
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e1")},
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e2")},
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e3"), IsCommon: true},
			}}
		close(haves)
	}()
	sr := &ServerResponse{req: &UploadPackRequest{UploadPackCommands: haves, UploadRequest: UploadRequest{Capabilities: capability.NewList()}}}
	b := bytes.NewBuffer(nil)
	err := sr.Encode(b)
	c.Assert(err, IsNil)

	c.Assert(b.String(), Equals, "0031ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e3\n")
}

func (s *ServerResponseSuite) TestEncodeSingleAckDone(c *C) {
	haves := make(chan UploadPackCommand)
	go func() {
		haves <- UploadPackCommand{
			Acks: []UploadPackRequestAck{
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e1")},
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e2")},
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e3"), IsCommon: true},
			},
			Done: true,
		}
		close(haves)
	}()
	sr := &ServerResponse{req: &UploadPackRequest{UploadPackCommands: haves, UploadRequest: UploadRequest{Capabilities: capability.NewList()}}}
	b := bytes.NewBuffer(nil)
	err := sr.Encode(b)
	c.Assert(err, IsNil)

	c.Assert(b.String(), Equals, "0031ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e3\n")
}

func (s *ServerResponseSuite) TestEncodeMutiAck(c *C) {
	haves := make(chan UploadPackCommand)
	go func() {
		haves <- UploadPackCommand{
			Acks: []UploadPackRequestAck{
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e1")},
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e2"), IsCommon: true, IsReady: true},
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e3")},
			},
		}
		haves <- UploadPackCommand{
			Acks: []UploadPackRequestAck{},
			Done: true,
		}
		close(haves)
	}()
	capabilities := capability.NewList()
	capabilities.Add(capability.MultiACK)
	sr := &ServerResponse{req: &UploadPackRequest{UploadPackCommands: haves, UploadRequest: UploadRequest{Capabilities: capabilities}}}
	b := bytes.NewBuffer(nil)
	err := sr.Encode(b)
	c.Assert(err, IsNil)

	lines := strings.Split(b.String(), "\n")
	c.Assert(len(lines), Equals, 5)
	c.Assert(lines[0], Equals, "003aACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e2 continue")
	c.Assert(lines[1], Equals, "003aACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e3 continue")
	c.Assert(lines[2], Equals, "0008NAK")
	c.Assert(lines[3], Equals, "0031ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e3")
	c.Assert(lines[4], Equals, "")
}

func (s *ServerResponseSuite) TestEncodeMutiAckOnlyOneNak(c *C) {
	haves := make(chan UploadPackCommand)
	go func() {
		haves <- UploadPackCommand{
			Acks: []UploadPackRequestAck{}, //no common hash
			Done: true,
		}
		close(haves)
	}()
	capabilities := capability.NewList()
	capabilities.Add(capability.MultiACK)
	sr := &ServerResponse{req: &UploadPackRequest{UploadPackCommands: haves, UploadRequest: UploadRequest{Capabilities: capabilities}}}
	b := bytes.NewBuffer(nil)
	err := sr.Encode(b)
	c.Assert(err, IsNil)

	lines := strings.Split(b.String(), "\n")
	c.Assert(len(lines), Equals, 2)
	c.Assert(lines[0], Equals, "0008NAK")
	c.Assert(lines[1], Equals, "")
}

func (s *ServerResponseSuite) TestEncodeMutiAckDetailed(c *C) {
	haves := make(chan UploadPackCommand)
	go func() {
		haves <- UploadPackCommand{
			Acks: []UploadPackRequestAck{
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e1")},
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e2"), IsCommon: true, IsReady: true},
				{Hash: plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e3"), IsCommon: true},
			},
		}
		haves <- UploadPackCommand{
			Acks: []UploadPackRequestAck{},
			Done: true,
		}
		close(haves)
	}()
	capabilities := capability.NewList()
	capabilities.Add(capability.MultiACKDetailed)
	sr := &ServerResponse{req: &UploadPackRequest{UploadPackCommands: haves, UploadRequest: UploadRequest{Capabilities: capabilities}}}
	b := bytes.NewBuffer(nil)
	err := sr.Encode(b)
	c.Assert(err, IsNil)

	lines := strings.Split(b.String(), "\n")
	c.Assert(len(lines), Equals, 5)
	c.Assert(lines[0], Equals, "0037ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e2 ready")
	c.Assert(lines[1], Equals, "0038ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e3 common")
	c.Assert(lines[2], Equals, "0008NAK")
	c.Assert(lines[3], Equals, "0031ACK 6ecf0ef2c2dffb796033e5a02219af86ec6584e3")
	c.Assert(lines[4], Equals, "")
}
