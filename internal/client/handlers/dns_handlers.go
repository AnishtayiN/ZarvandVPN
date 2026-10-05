// ==============================================================================
// ZarvandVPN
// Author: ZarvandVPN Project
// Github: https://github.com/AnishtayiN
// Year: 2026
// ==============================================================================
package handlers

import (
	Enums "zarvand/internal/enums"
	VpnProto "zarvand/internal/vpnproto"
	"net"
)

func init() {
	RegisterHandler(Enums.PACKET_DNS_QUERY_REQ_ACK, handleDNSQueryAck)
	RegisterHandler(Enums.PACKET_DNS_QUERY_RES, handleDNSQueryRes)
}

func handleDNSQueryAck(c ClientContext, packet VpnProto.Packet, addr *net.UDPAddr) error {
	return c.HandleDNSQueryAck(packet)
}

func handleDNSQueryRes(c ClientContext, packet VpnProto.Packet, addr *net.UDPAddr) error {
	return c.HandleDNSQueryRes(packet)
}
