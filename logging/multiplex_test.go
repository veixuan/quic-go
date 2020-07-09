package logging

import (
	"crypto/rand"
	"net"
	"time"

	"github.com/lucas-clemente/quic-go/internal/wire"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tracing", func() {
	Context("Tracer", func() {
		var (
			tracer   Tracer
			tr1, tr2 *MockTracer
		)

		BeforeEach(func() {
			tr1 = NewMockTracer(mockCtrl)
			tr2 = NewMockTracer(mockCtrl)
			tracer = NewMultiplexedTracer(tr1, tr2)
		})

		It("multiplexes the TracerForServer call", func() {
			tr1.EXPECT().TracerForServer(ConnectionID{1, 2, 3})
			tr2.EXPECT().TracerForServer(ConnectionID{1, 2, 3})
			tracer.TracerForServer(ConnectionID{1, 2, 3})
		})

		It("multiplexes the TracerForClient call", func() {
			tr1.EXPECT().TracerForClient(ConnectionID{1, 2, 3})
			tr2.EXPECT().TracerForClient(ConnectionID{1, 2, 3})
			tracer.TracerForClient(ConnectionID{1, 2, 3})
		})

		It("uses multiple connection tracers", func() {
			ctr1 := NewMockConnectionTracer(mockCtrl)
			ctr2 := NewMockConnectionTracer(mockCtrl)
			tr1.EXPECT().TracerForClient(ConnectionID{1, 2, 3}).Return(ctr1)
			tr2.EXPECT().TracerForClient(ConnectionID{1, 2, 3}).Return(ctr2)
			tr := tracer.TracerForClient(ConnectionID{1, 2, 3})
			ctr1.EXPECT().LossTimerCanceled()
			ctr2.EXPECT().LossTimerCanceled()
			tr.LossTimerCanceled()
		})

		It("handles tracers that return a nil ConnectionTracer", func() {
			ctr1 := NewMockConnectionTracer(mockCtrl)
			tr1.EXPECT().TracerForClient(ConnectionID{1, 2, 3}).Return(ctr1)
			tr2.EXPECT().TracerForClient(ConnectionID{1, 2, 3})
			tr := tracer.TracerForClient(ConnectionID{1, 2, 3})
			ctr1.EXPECT().LossTimerCanceled()
			tr.LossTimerCanceled()
		})

		It("returns nil when all tracers return a nil ConnectionTracer", func() {
			tr1.EXPECT().TracerForClient(ConnectionID{1, 2, 3})
			tr2.EXPECT().TracerForClient(ConnectionID{1, 2, 3})
			Expect(tracer.TracerForClient(ConnectionID{1, 2, 3})).To(BeNil())
		})
	})

	Context("Connection Tracer", func() {
		var (
			tracer   ConnectionTracer
			tr1, tr2 *MockConnectionTracer
		)

		BeforeEach(func() {
			tr1 = NewMockConnectionTracer(mockCtrl)
			tr2 = NewMockConnectionTracer(mockCtrl)
			tracer = newConnectionMultiplexer(tr1, tr2)
		})

		It("multiplexes the ConnectionStarted event", func() {
			local := &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4)}
			remote := &net.UDPAddr{IP: net.IPv4(4, 3, 2, 1)}
			tr1.EXPECT().StartedConnection(local, remote, VersionNumber(1234), ConnectionID{1, 2, 3, 4}, ConnectionID{4, 3, 2, 1})
			tr2.EXPECT().StartedConnection(local, remote, VersionNumber(1234), ConnectionID{1, 2, 3, 4}, ConnectionID{4, 3, 2, 1})
			tracer.StartedConnection(local, remote, VersionNumber(1234), ConnectionID{1, 2, 3, 4}, ConnectionID{4, 3, 2, 1})
		})

		It("multiplexes the ClosedConnection event", func() {
			tr1.EXPECT().ClosedConnection(CloseReasonHandshakeTimeout)
			tr2.EXPECT().ClosedConnection(CloseReasonHandshakeTimeout)
			tracer.ClosedConnection(CloseReasonHandshakeTimeout)
		})

		It("multiplexes the SentTransportParameters event", func() {
			tp := &wire.TransportParameters{InitialMaxData: 1337}
			tr1.EXPECT().SentTransportParameters(tp)
			tr2.EXPECT().SentTransportParameters(tp)
			tracer.SentTransportParameters(tp)
		})

		It("multiplexes the ReceivedTransportParameters event", func() {
			tp := &wire.TransportParameters{InitialMaxData: 1337}
			tr1.EXPECT().ReceivedTransportParameters(tp)
			tr2.EXPECT().ReceivedTransportParameters(tp)
			tracer.ReceivedTransportParameters(tp)
		})

		It("multiplexes the SentPacket event", func() {
			hdr := &ExtendedHeader{Header: Header{DestConnectionID: ConnectionID{1, 2, 3}}}
			ack := &AckFrame{AckRanges: []AckRange{{Smallest: 1, Largest: 10}}}
			ping := &PingFrame{}
			tr1.EXPECT().SentPacket(hdr, ByteCount(1337), ack, []Frame{ping})
			tr2.EXPECT().SentPacket(hdr, ByteCount(1337), ack, []Frame{ping})
			tracer.SentPacket(hdr, 1337, ack, []Frame{ping})
		})

		It("multiplexes the ReceivedVersionNegotiationPacket event", func() {
			hdr := &Header{DestConnectionID: ConnectionID{1, 2, 3}}
			tr1.EXPECT().ReceivedVersionNegotiationPacket(hdr)
			tr2.EXPECT().ReceivedVersionNegotiationPacket(hdr)
			tracer.ReceivedVersionNegotiationPacket(hdr)
		})

		It("multiplexes the ReceivedRetry event", func() {
			hdr := &Header{DestConnectionID: ConnectionID{1, 2, 3}}
			tr1.EXPECT().ReceivedRetry(hdr)
			tr2.EXPECT().ReceivedRetry(hdr)
			tracer.ReceivedRetry(hdr)
		})

		It("multiplexes the ReceivedPacket event", func() {
			hdr := &ExtendedHeader{Header: Header{DestConnectionID: ConnectionID{1, 2, 3}}}
			ping := &PingFrame{}
			tr1.EXPECT().ReceivedPacket(hdr, ByteCount(1337), []Frame{ping})
			tr2.EXPECT().ReceivedPacket(hdr, ByteCount(1337), []Frame{ping})
			tracer.ReceivedPacket(hdr, 1337, []Frame{ping})
		})

		It("multiplexes the ReceivedStatelessResetToken event", func() {
			var token [16]byte
			rand.Read(token[:])
			tr1.EXPECT().ReceivedStatelessReset(&token)
			tr2.EXPECT().ReceivedStatelessReset(&token)
			tracer.ReceivedStatelessReset(&token)
		})

		It("multiplexes the BufferedPacket event", func() {
			tr1.EXPECT().BufferedPacket(PacketTypeHandshake)
			tr2.EXPECT().BufferedPacket(PacketTypeHandshake)
			tracer.BufferedPacket(PacketTypeHandshake)
		})

		It("multiplexes the DroppedPacket event", func() {
			tr1.EXPECT().DroppedPacket(PacketTypeInitial, ByteCount(1337), PacketDropHeaderParseError)
			tr2.EXPECT().DroppedPacket(PacketTypeInitial, ByteCount(1337), PacketDropHeaderParseError)
			tracer.DroppedPacket(PacketTypeInitial, 1337, PacketDropHeaderParseError)
		})

		It("multiplexes the UpdatedMetrics event", func() {
			rttStats := &RTTStats{}
			rttStats.UpdateRTT(time.Second, 0, time.Now())
			tr1.EXPECT().UpdatedMetrics(rttStats, ByteCount(1337), ByteCount(42), 13)
			tr2.EXPECT().UpdatedMetrics(rttStats, ByteCount(1337), ByteCount(42), 13)
			tracer.UpdatedMetrics(rttStats, 1337, 42, 13)
		})

		It("multiplexes the LostPacket event", func() {
			tr1.EXPECT().LostPacket(EncryptionHandshake, PacketNumber(42), PacketLossReorderingThreshold)
			tr2.EXPECT().LostPacket(EncryptionHandshake, PacketNumber(42), PacketLossReorderingThreshold)
			tracer.LostPacket(EncryptionHandshake, 42, PacketLossReorderingThreshold)
		})

		It("multiplexes the UpdatedPTOCount event", func() {
			tr1.EXPECT().UpdatedPTOCount(uint32(88))
			tr2.EXPECT().UpdatedPTOCount(uint32(88))
			tracer.UpdatedPTOCount(88)
		})

		It("multiplexes the UpdatedKeyFromTLS event", func() {
			tr1.EXPECT().UpdatedKeyFromTLS(EncryptionHandshake, PerspectiveClient)
			tr2.EXPECT().UpdatedKeyFromTLS(EncryptionHandshake, PerspectiveClient)
			tracer.UpdatedKeyFromTLS(EncryptionHandshake, PerspectiveClient)
		})

		It("multiplexes the UpdatedKey event", func() {
			tr1.EXPECT().UpdatedKey(KeyPhase(42), true)
			tr2.EXPECT().UpdatedKey(KeyPhase(42), true)
			tracer.UpdatedKey(KeyPhase(42), true)
		})

		It("multiplexes the DroppedEncryptionLevel event", func() {
			tr1.EXPECT().DroppedEncryptionLevel(EncryptionHandshake)
			tr2.EXPECT().DroppedEncryptionLevel(EncryptionHandshake)
			tracer.DroppedEncryptionLevel(EncryptionHandshake)
		})

		It("multiplexes the SetLossTimer event", func() {
			now := time.Now()
			tr1.EXPECT().SetLossTimer(TimerTypePTO, EncryptionHandshake, now)
			tr2.EXPECT().SetLossTimer(TimerTypePTO, EncryptionHandshake, now)
			tracer.SetLossTimer(TimerTypePTO, EncryptionHandshake, now)
		})

		It("multiplexes the LossTimerExpired event", func() {
			tr1.EXPECT().LossTimerExpired(TimerTypePTO, EncryptionHandshake)
			tr2.EXPECT().LossTimerExpired(TimerTypePTO, EncryptionHandshake)
			tracer.LossTimerExpired(TimerTypePTO, EncryptionHandshake)
		})

		It("multiplexes the LossTimerCanceled event", func() {
			tr1.EXPECT().LossTimerCanceled()
			tr2.EXPECT().LossTimerCanceled()
			tracer.LossTimerCanceled()
		})

		It("multiplexes the Close event", func() {
			tr1.EXPECT().Close()
			tr2.EXPECT().Close()
			tracer.Close()
		})
	})
})
