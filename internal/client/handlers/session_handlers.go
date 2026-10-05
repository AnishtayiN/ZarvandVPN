// ==============================================================================
// ZarvandVPN
// Author: ZarvandVPN Project
// Github: https://github.com/AnishtayiN
// Year: 2026
// ==============================================================================
package handlers

import (
	"net"

	Enums "zarvand/internal/enums"
	VpnProto "zarvand/internal/vpnproto"
)

func init() {
	RegisterHandler(Enums.PACKET_SESSION_BUSY, handleSessionBusy)
	RegisterHandler(Enums.PACKET_ERROR_DROP, handleErrorDrop)
}

func handleSessionBusy(c ClientContext, packet VpnProto.Packet, addr *net.UDPAddr) error {
	return c.HandleSessionBusy()
}

func handleErrorDrop(c ClientContext, packet VpnProto.Packet, addr *net.UDPAddr) error {
	return c.HandleErrorDrop(packet)
}
